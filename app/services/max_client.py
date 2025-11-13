import logging
from typing import Any, Dict, Optional

from maxapi import Bot
from maxapi.enums.update import UpdateType
from maxapi.methods.types.getted_updates import process_update_webhook
from maxapi.types import MessageCreated, UpdateUnion

from app.config import settings
from app.models.message import BotResponse

logger = logging.getLogger(__name__)


class MaxClient:
    """Обертка над maxapi.Bot для работы с сообщениями и вебхуками."""

    def __init__(self) -> None:
        self.logger = logger
        self.bot = Bot(token=settings.MAX_BOT_TOKEN)

    async def close(self) -> None:
        """Закрытие HTTP-сессий maxapi."""
        await self.bot.close_session()

    async def send_message(self, message: BotResponse) -> bool:
        """Отправка текстового сообщения пользователю."""
        try:
            if settings.DEBUG:
                self.logger.info(
                    "[DEV] Отправка сообщения в чат %s: %s",
                    message.chat_id,
                    message.text,
                )

            chat_id = int(message.chat_id)
            payload: Dict[str, Any] = {
                "chat_id": chat_id,
                "text": message.text,
            }

            # MAX API не поддерживает Markdown напрямую, поэтому опускаем parse_mode
            if message.reply_to_message_id:
                payload["reply_to_message_id"] = message.reply_to_message_id

            result = await self.bot.send_message(**payload)
            self.logger.info("Сообщение отправлено в чат %s", message.chat_id)
            return result is not None
        except Exception as exc:  # pragma: no cover - сетевые ошибки
            self.logger.error("Ошибка при отправке сообщения: %s", exc, exc_info=True)
            return False

    async def set_webhook(self, url: str) -> bool:
        """Установка webhook на стороне MAX."""
        try:
            if settings.DEBUG:
                self.logger.info("[DEV] Установка вебхука: %s", url)
                return True

            await self.delete_webhook()
            await self.bot.subscribe_webhook(
                url=url,
                update_types=[UpdateType.MESSAGE_CREATED],
                secret=settings.MAX_WEBHOOK_SECRET,
            )
            self.logger.info("Webhook успешно установлен: %s", url)
            return True
        except Exception as exc:  # pragma: no cover
            self.logger.error("Ошибка настройки вебхука: %s", exc, exc_info=True)
            return False

    async def delete_webhook(self) -> None:
        """Удаление всех активных вебхуков."""
        try:
            await self.bot.delete_webhook()
        except Exception as exc:  # pragma: no cover
            self.logger.warning("Не удалось удалить вебхук: %s", exc, exc_info=True)

    async def parse_webhook(self, data: Dict[str, Any]) -> Optional[MessageCreated]:
        """
        Преобразует сырой payload от MAX в python-объект события.

        Возвращает событие MessageCreated или None, если событие не поддерживается.
        """
        try:
            event: UpdateUnion = await process_update_webhook(event_json=data, bot=self.bot)
            if isinstance(event, MessageCreated):
                return event
            self.logger.debug("Получено неподдерживаемое событие: %s", type(event))
            return None
        except Exception as exc:  # pragma: no cover
            self.logger.error("Ошибка обработки вебхука: %s", exc, exc_info=True)
            return None