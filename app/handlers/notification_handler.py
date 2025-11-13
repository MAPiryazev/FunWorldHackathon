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
            self.logger.info("получено уведомление для пользователя")

            message_text = self._format_notification_message(notification)

            response = BotResponse(
                chat_id=str(notification.user_id),
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
            f"{category_info}\n"
            f"📝 {notification.text}\n"
            f"{time_info}\n"
            f"{priority_info}\n"
            f"{complexity_info}"
        )

        if notification.notes:
            message += f"\n📋 **Заметки:** {notification.notes}"

        message += f"\n\n🆔 ID: {notification.id[:8]}..."

        return message



