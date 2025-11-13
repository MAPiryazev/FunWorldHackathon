from app.models.message import BotResponse
from app.services.state_manager import StateManager
from app.utils.formatters import format_user_tasks
from app.services.reminder_client import ReminderClient

class CommandHandler:

    def __init__(self, state_manager: StateManager):
        self.state_manager = state_manager

    async def handle_start(self, user_id:str) -> BotResponse:

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

        return BotResponse(chat_id="", text=text)

    async def handle_help(self) -> BotResponse:
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

        return BotResponse(chat_id="", text=text)

    async def handle_cancel(self, user_id:str) -> BotResponse:
        await self.state_manager.clear_state(user_id)

        text = "❌ Текущее действие отменено. Что хочешь сделать?"
        return BotResponse(chat_id="", text=text)


    async def _handle_tasks(self, user_id: str) -> BotResponse:
        """Обработка команды просмотра задач"""
        try:
            self.logger.info(f"Просмотр задач для пользователя {user_id}")

            # Получаем задачи пользователя из reminder-service
            tasks = await ReminderClient.get_user_tasks(user_id)

            if tasks is None:
                return BotResponse(
                    text="❌ Сервис задач временно недоступен. Попробуй позже."
                )

            # Проверяем есть ли задачи
            if not tasks:
                return BotResponse(
                    text="📭 У тебя пока нет напоминаний"
                )

            # Форматируем список задач
            task_list = format_user_tasks(tasks)

            return BotResponse(
                user_id=user_id,
                text=task_list
            )

        except Exception as e:
            self.logger.error(f"Ошибка обработки команды задач: {e}")
            return BotResponse(
                user_id=user_id,
                text="❌ Не удалось загрузить список задач. Попробуй позже."
            )
