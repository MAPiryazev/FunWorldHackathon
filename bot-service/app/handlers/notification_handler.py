import logging

from app.models.notification import NotificationRequest
from app.models.message import BotResponse
from app.services.max_client import MaxClient
from app.utils.formatters import format_datetime, format_priority, format_category, format_complexity

logger = logging.getLogger(__name__)

class NotificationHandler:
    def __init__(self, max_client: MaxClient):
        self.max_client = max_client
        self.logger = logger

    async def handle_notification(self, notification: NotificationRequest) -> bool:

        try:
            self.logger.info("получено уведомление для пользователя %s", notification.user_id)

            # Используем chat_id из уведомления
            chat_id = notification.chat_id if notification.chat_id else str(notification.user_id)
            if notification.chat_id:
                self.logger.info("Используется chat_id из уведомления: %s", chat_id)

            message_text = self._format_notification_message(notification)

            response = BotResponse(
                chat_id=chat_id,
                user_id=str(notification.user_id),
                text=message_text,
            )

            success = await self.max_client.send_message(response)

            if success:
                self.logger.info("уведомление отправлено")

            else:
                self.logger.error("не удалось отправить уведомление")

            return success

        except Exception as e:
            self.logger.error("ошибка обработки уведомления", exc_info=True)
            return False

    def _format_notification_message(self, notification: NotificationRequest) -> str:
        """Форматирование текста уведомления для пользователя"""

        # Используем функции из formatters
        time_info = format_datetime(notification.remind_at)
        priority_info = format_priority(notification.priority)
        category_info = format_category(notification.category)
        complexity_info = format_complexity(notification.complexity)

        message = (
            f"🔔 **НАПОМИНАНИЕ**\n\n"
            f"📝 {notification.text}\n\n"
            f"🗓️ Напомнить: {time_info}\n"
            f"Категория: {category_info}\n"     
            f"Приоритет: {priority_info}\n"
            f"Сложность: {complexity_info}\n\n"
            f"🆔 ID напоминания: {notification.id}"
        )

        if notification.notes:
            message += f"\n📋 **Заметки:** {notification.notes}"

        return message



