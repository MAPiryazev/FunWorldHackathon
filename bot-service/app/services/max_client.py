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

            # Валидация: chat_id должен быть числом
            try:
                chat_id = int(message.chat_id)
            except (ValueError, TypeError):
                self.logger.error(
                    "Некорректный chat_id (должен быть числом): %s (тип: %s)",
                    message.chat_id,
                    type(message.chat_id).__name__,
                )
                return False
            payload: Dict[str, Any] = {
                "chat_id": chat_id,
                "text": message.text,
            }
            
            # Логируем содержимое сообщения для отладки
            self.logger.debug(
                "Отправка сообщения в MAX API: chat_id=%s, text_length=%d, text_preview=%s",
                chat_id,
                len(message.text),
                message.text[:100] + "..." if len(message.text) > 100 else message.text,
            )

            if message.reply_to_message_id:
                payload["reply_to_message_id"] = message.reply_to_message_id
            
            result = await self.bot.send_message(**payload)
            
            
            # MAX API возвращает SendedMessage с Message внутри
            # message_id находится в result.body.mid или result.message.body.mid
            message_id = None
            if hasattr(result, 'message') and hasattr(result.message, 'body') and hasattr(result.message.body, 'mid'):
                message_id = result.message.body.mid
            elif hasattr(result, 'body') and hasattr(result.body, 'mid'):
                message_id = result.body.mid
            elif hasattr(result, 'message_id'):
                message_id = result.message_id
            
            if message_id:
                self.logger.info(
                    "Сообщение успешно отправлено в чат %s, message_id=%s",
                    message.chat_id,
                    message_id,
                )
            else:
                self.logger.warning(
                    "Не удалось извлечь message_id из результата отправки (тип: %s): %s",
                    type(result).__name__,
                    result,
                )
            
            return True
        except Exception as exc:  
            self.logger.error("Ошибка при отправке сообщения: %s", exc, exc_info=True)
            return False

    async def set_webhook(self, url: str) -> bool:
        """Установка webhook на стороне MAX."""
        try:
            if settings.DEBUG:
                self.logger.info("[DEV] Установка вебхука (боевой вызов): %s", url)

            await self.delete_webhook()
            update_types = [UpdateType.MESSAGE_CREATED]
            
            await self.bot.subscribe_webhook(
                url=url,
                update_types=update_types,
                secret=settings.MAX_WEBHOOK_SECRET,
            )
            self.logger.info("Webhook успешно установлен: %s", url)
            return True
        except Exception as exc:  
            self.logger.error("Ошибка настройки вебхука: %s", exc, exc_info=True)
            return False

    async def delete_webhook(self) -> None:
        """Удаление всех активных вебхуков."""
        try:
            await self.bot.delete_webhook()
        except Exception as exc:  
            self.logger.warning("Не удалось удалить вебхук: %s", exc, exc_info=True)

    async def parse_webhook(self, data: Dict[str, Any]) -> Optional[UpdateUnion]:
        """
        Преобразует сырой payload от MAX в python-объект события.

        Возвращает событие (MessageCreated, CallbackCreated и т.д.) или None, если событие не поддерживается.
        """
        try:
            event: UpdateUnion = await process_update_webhook(event_json=data, bot=self.bot)
            return event
        except Exception as exc:  
            self.logger.error("Ошибка обработки вебхука: %s", exc, exc_info=True)
            return None