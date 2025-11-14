import logging
from app.models.message import BotResponse
from app.models.user import UserState
from app.services.state_manager import StateManager
from app.services.reminder_client import ReminderClient
from app.utils.formatters import format_datetime, format_priority, format_category, format_complexity

logger = logging.getLogger(__name__)


class ReminderHandler:
    """Обработчик создания напоминаний"""

    def __init__(self, state_manager: StateManager, reminder_client: ReminderClient):
        self.state_manager = state_manager
        self.reminder_client = reminder_client
        self.logger = logger

    async def handle_reminder_creation(self, user_id: str, chat_id: str, nlp_data: dict) -> BotResponse:
        """Обработка создания напоминания на основе данных от NLP"""
        try:
            # Подготавливаем данные для reminder-service
            task_data = {
                "user_id": user_id,
                "chat_id": chat_id,  # ID чата для отправки уведомления
                "text": nlp_data["text"],
                "remind_at": nlp_data["remind_at"],
                "complexity": nlp_data.get("complexity", 1),
                "priority": nlp_data.get("priority", "medium"),
                "category": nlp_data.get("category", "другие"),
                "notes": nlp_data.get("notes", "")
            }

            # Отправляем в reminder-service
            result = await self.reminder_client.create_task(task_data)

            if result:
                await self.state_manager.clear_state(user_id)

                # Формируем красивый ответ
                remind_at = nlp_data.get("remind_at")
                time_info = format_datetime(remind_at)
                priority_emoji = format_priority(nlp_data.get("priority", "medium"))
                category_emoji = format_category(nlp_data.get("category", "другие"))
                complexity_emoji = format_complexity(nlp_data.get("complexity", 1))

                text = (
                    f"✅ Напоминание создано!\n\n"
                    f"📝 {nlp_data['text']}\n\n"
                    f"🗓️ Напомнить: {time_info}\n"
                    f"Категория: {category_emoji}\n"     
                    f"Приоритет: {priority_emoji}\n"
                    f"Сложность: {complexity_emoji}\n\n"
                    f"🆔 ID напоминания: {result['id']}\n"
                )

                if nlp_data.get('notes'):
                    text += f"\n📋 Заметки: {nlp_data['notes']}\n"

            else:
                text = "❌ Не удалось создать напоминание. Попробуй еще раз позже."

            return BotResponse(chat_id=chat_id, text=text)

        except Exception as e:
            self.logger.error(f"Ошибка создания напоминания: {e}")
            return BotResponse(
                chat_id=chat_id,
                text="❌ Произошла ошибка при создании напоминания. Попробуй еще раз.",
            )

    async def handle_time_request(self, user_id: str, chat_id: str, text: str) -> BotResponse:
        """Запрос времени у пользователя"""
        await self.state_manager.set_user_state(
            user_id,
            UserState.AWAITING_TIME,
            {"pending_reminder": text}
        )


        return BotResponse(
            chat_id=chat_id,
            text="⏰ Уточни, пожалуйста, когда нужно напомнить?  (например: 'завтра в 15:00' или 'сегодня вечером'):",
        )


