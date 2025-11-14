import logging
from app.models.message import BotResponse
from app.services.state_manager import StateManager
from app.utils.formatters import format_user_tasks
from app.services.reminder_client import ReminderClient

logger = logging.getLogger(__name__)

class CommandHandler:

    def __init__(self, state_manager: StateManager, reminder_client: ReminderClient):
        self.state_manager = state_manager
        self.reminder_client = reminder_client
        self.logger = logger

    async def handle_start(self, user_id: str, chat_id: str) -> BotResponse:

        await self.state_manager.clear_state(user_id)

        text = (
            "👋 Привет! Я твой персональный ИИ тайм-менеджер.\n\n"
            "Я могу помочь тебе:\n"
            "• Создать напоминание по времени\n"
            "• Установить приоритет задач\n"
            "• Организовать твои дела\n\n"
            "Просто напиши мне что-то вроде:\n"
            "• \"Напомни завтра в 10:00 позвонить маме\"\n"
            "• \"Напомни мне о встрече в пятницу\"\n\n"
            "Используй /help для списка команд"
        )

        return BotResponse(chat_id=chat_id, text=text)

    async def handle_help(self, chat_id: str) -> BotResponse:
        """Обработка команды /help"""
        text = (
            "📋 Доступные команды:\n\n"
            "/start - начать работу с ботом\n"
            "/help - показать эту справку\n"
            "/tasks - показать все задачи\n"
            "/cancel - отменить текущее действие\n\n"
            "Примеры напоминаний:\n"
            "• \"Позвонить врачу завтра в 15:00\"\n"
            "• \"Купить продукты сегодня вечером\"\n"
            "• \"Встреча с коллегой в 14:30\"\n"
        )

        return BotResponse(chat_id=chat_id, text=text)

    async def handle_cancel(self, user_id: str, chat_id: str) -> BotResponse:
        await self.state_manager.clear_state(user_id)

        text = "❌ Текущее действие отменено. Что хочешь сделать?"
        return BotResponse(chat_id=chat_id, text=text)


    async def _handle_tasks(self, user_id: str, chat_id: str) -> BotResponse:
        """Обработка команды просмотра задач"""
        try:
            self.logger.info(f"Просмотр задач для пользователя {user_id}")

            # Получаем задачи пользователя из reminder-service
            tasks = await self.reminder_client.get_user_tasks(user_id)

            if tasks is None:
                return BotResponse(
                    chat_id=chat_id,
                    text="❌ Сервис задач временно недоступен. Попробуй позже.",
                )

            # Проверяем есть ли задачи
            if not tasks:
                return BotResponse(
                    chat_id=chat_id,
                    text="📭 У тебя пока нет напоминаний",
                )

            # Форматируем список задач
            self.logger.info(f"Получено задач: {len(tasks)}, данные: {tasks}")
            task_list = format_user_tasks(tasks)
            self.logger.info(f"Отформатированный текст: {task_list[:200]}...")  # Первые 200 символов

            return BotResponse(
                chat_id=chat_id,
                user_id=user_id,
                text=task_list,
            )

        except Exception as e:
            self.logger.error(f"Ошибка обработки команды задач: {e}")
            return BotResponse(
                chat_id=chat_id,
                user_id=user_id,
                text="❌ Не удалось загрузить список задач. Попробуй позже.",
            )
