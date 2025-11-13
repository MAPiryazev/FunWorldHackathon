import asyncio
import logging
from contextlib import asynccontextmanager
from typing import Optional

from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse
from maxapi.types import MessageCreated

from app.config import settings
from app.handlers.message_handler import MessageHandler
from app.handlers.notification_handler import NotificationHandler
from app.handlers.reminder_handler import ReminderHandler
from app.models.message import BotResponse, MaxMessage
from app.models.notification import NotificationRequest
from app.services.max_client import MaxClient
from app.services.nlp_client import NLPClient
from app.services.reminder_client import ReminderClient
from app.services.state_manager import StateManager
from app.services.tuna_manager import TunaTunnel

logging.basicConfig(
    level=logging.DEBUG if settings.DEBUG else logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)

logger = logging.getLogger(__name__)

# Глобальные объекты
state_manager: Optional[StateManager] = None
reminder_client: Optional[ReminderClient] = None
nlp_client: Optional[NLPClient] = None
max_client: Optional[MaxClient] = None
reminder_handler: Optional[ReminderHandler] = None
message_handler: Optional[MessageHandler] = None
notification_handler: Optional[NotificationHandler] = None
tuna_tunnel: Optional[TunaTunnel] = None
webhook_url: Optional[str] = None


def _max_event_to_message(event: MessageCreated) -> MaxMessage:
    """Преобразование события MAX во внутреннюю модель сообщения."""
    chat_id = event.message.recipient.chat_id or event.message.recipient.user_id
    sender = event.message.sender
    body = event.message.body

    if chat_id is None:
        raise ValueError("Не удалось определить идентификатор чата для события MAX")

    return MaxMessage(
        message_id=str(body.mid),
        user_id=str(sender.user_id),
        chat_id=str(chat_id),
        text=(body.text or "").strip(),
        timestamp=int(event.message.timestamp),
        username=sender.username,
        first_name=sender.first_name,
    )


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Управление жизненным циклом приложения."""
    global state_manager, reminder_client, nlp_client, max_client
    global reminder_handler, message_handler, notification_handler, tuna_tunnel
    global webhook_url

    logger.info("Запуск MAX Bot Service")

    state_manager = StateManager()
    reminder_client = ReminderClient()
    nlp_client = NLPClient()
    max_client = MaxClient()
    reminder_handler = ReminderHandler(state_manager, reminder_client)
    message_handler = MessageHandler(nlp_client, state_manager, reminder_handler)
    notification_handler = NotificationHandler(max_client)
    tuna_tunnel = TunaTunnel()

    tunnel_url = await tuna_tunnel.start()
    if tunnel_url:
        webhook_url = f"{tunnel_url.rstrip('/')}{settings.WEBHOOK_PATH}"
        if not await max_client.set_webhook(webhook_url):
            logger.warning("Не удалось установить webhook по адресу %s", webhook_url)
    else:
        webhook_url = None
        logger.warning(
            "Webhook URL не определен. Запустите tuna вручную и задайте TUNA_PUBLIC_URL."
        )

    try:
        yield
    finally:
        if max_client:
            await max_client.delete_webhook()
            await max_client.close()
        if tuna_tunnel:
            await tuna_tunnel.stop()


app = FastAPI(
    title="MAX Bot Service",
    version="0.2.0",
    description="MVP интеграция бота с мессенджером MAX через Tuna tunnel",
    lifespan=lifespan,
)


@app.get("/health")
async def health_check() -> dict:
    """Простая проверка состояния сервиса."""
    payload = {"status": "ok"}
    if webhook_url:
        payload["webhook"] = webhook_url
    return payload


@app.post("/notifications")
async def handle_notification(notification: NotificationRequest) -> JSONResponse:
    """Получение уведомлений от внешнего сервиса напоминаний."""
    if not notification_handler:
        raise HTTPException(status_code=503, detail="Notification handler is not ready")

    success = await notification_handler.handle_notification(notification)

    if not success:
        raise HTTPException(
            status_code=502, detail="Не удалось отправить уведомление пользователю"
        )

    return JSONResponse({"status": "delivered"})


@app.post(settings.WEBHOOK_PATH)
async def max_webhook(request: Request) -> JSONResponse:
    """Webhook MAX: принимает входящие сообщения от пользователей."""
    if not (max_client and message_handler):
        raise HTTPException(status_code=503, detail="Бот еще не готов")

    payload = await request.json()
    event = await max_client.parse_webhook(payload)

    if not event:
        logger.debug("Пропускаем событие без обработки")
        return JSONResponse({"ok": True})

    try:
        user_message = _max_event_to_message(event)
    except ValueError as exc:
        logger.error("Некорректное событие MAX: %s", exc)
        return JSONResponse({"ok": False})
    logger.info(
        "Сообщение от %s: %s",
        user_message.user_id,
        user_message.text[:80],
    )

    bot_response: BotResponse = await message_handler.handle_message(user_message)
    bot_response.chat_id = user_message.chat_id
    bot_response.user_id = user_message.user_id

    success = await max_client.send_message(bot_response)
    if not success:
        logger.error("Не удалось отправить ответ пользователю %s", user_message.user_id)

    return JSONResponse({"ok": success})
