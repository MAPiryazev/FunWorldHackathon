import logging
import re

from app.models.message import MaxMessage, BotResponse
from app.models.user import UserState
from app.services.nlp_client import NLPClient
from app.services.state_manager import StateManager
from app.handlers.command_handler import CommandHandler
from app.handlers.reminder_handler import ReminderHandler


logger = logging.getLogger(__name__)


class MessageHandler:
    """Основной обработчик входящих сообщений"""

    def __init__(self, nlp_client: NLPClient, state_manager: StateManager, reminder_handler: ReminderHandler):
        self.nlp_client = nlp_client
        self.state_manager = state_manager
        self.reminder_handler = reminder_handler
        self.reminder_client = reminder_handler.reminder_client
        self.command_handler = CommandHandler(state_manager)

        # Регулярные выражения для команд
        self.command_pattern = re.compile(r'^/([a-zA-Z0-9_]+)')
        self.logger = logger

    async def handle_message(self, message: MaxMessage) -> BotResponse:
        """Основной метод обработки входящего сообщения"""
        user_id = message.user_id
        chat_id = message.chat_id
        text = message.text.strip()

        self.logger.info(f"Обработка сообщения от {user_id}: '{text}'")

        # 1. Обработка команд (начинаются с /)
        if text.startswith('/'):
            response = await self._handle_command(user_id, text)
            response.chat_id = chat_id
            return response

        # 2. Проверка состояния пользователя
        user_state = await self.state_manager.get_user_state(user_id)

        # 3. Если пользователь ожидает указания времени
        if user_state == UserState.AWAITING_TIME:
            response = await self._handle_time_response(user_id, text)
            response.chat_id = chat_id
            return response

        # 4. Обработка обычного текстового сообщения
        response = await self._handle_text_message(user_id, text)
        response.chat_id = chat_id
        return response

    async def _handle_command(self, user_id: str, text: str) -> BotResponse:
        """Обработка текстовых команд"""
        try:
            # Извлекаем команду (убираем /)
            command_match = self.command_pattern.match(text)
            if not command_match:
                return BotResponse(
                    text="❌ Неверный формат команды. Используй /help для справки."
                )

            command = command_match.group(1).lower()
            self.logger.info(f"⌨️ Обработка команды '{command}' от пользователя {user_id}")

            # Обработка различных команд
            if command in ["start", "начать"]:
                return await self.command_handler.handle_start(user_id)

            elif command in ["help", "помощь"]:
                return await self.command_handler.handle_help(user_id)

            elif command in ["cancel", "отмена"]:
                return await self.command_handler.handle_cancel(user_id)

            elif command in ["tasks", "задачи"]:
                return await self.command_handler._handle_tasks(user_id)

            else:
                return BotResponse(
                    text=f"❌ Неизвестная команда: /{command}\nИспользуй /help для списка команд."
                )

        except Exception as e:
            self.logger.error(f"Ошибка обработки команды: {e}")
            return BotResponse(
                text="❌ Произошла ошибка при обработке команды. Попробуй еще раз."
            )

    async def _handle_text_message(self, user_id: str, text: str) -> BotResponse:
        """Обработка обычного текстового сообщения"""
        try:
            self.logger.info(f"Анализ текста от {user_id}: '{text}'")

            # Отправляем текст в NLP сервис для анализа
            nlp_result = await self.nlp_client.parse_text(text, user_id)

            if not nlp_result:
                return BotResponse(
                    text="❌ Сервис анализа временно недоступен. Попробуй позже."
                )

            # Проверяем результат парсинга
            if nlp_result.get("remind_at"):
                # NLP нашел время - создаем напоминание
                self.logger.info(f"NLP извлек данные: {nlp_result}")
                return await self.reminder_handler.handle_reminder_creation(user_id, nlp_result)
            else:
                # Время не указано - запрашиваем у пользователя
                self.logger.info("Время не указано, запрашиваем у пользователя")
                return await self.reminder_handler.handle_time_request(user_id, text)

        except Exception as e:
            self.logger.error(f"Ошибка обработки текстового сообщения: {e}")
            return BotResponse(
                text="❌ Произошла ошибка при обработке сообщения. Попробуй еще раз."
            )

    async def _handle_time_response(self, user_id: str, text: str) -> BotResponse:
        """Обработка ответа пользователя с указанием времени"""
        try:
            # Получаем контекст из состояния пользователя
            context = await self.state_manager.get_user_context(user_id)
            pending_reminder = context.get("pending_reminder") if context else None

            if pending_reminder:
                # Объединяем оригинальный текст с указанным временем
                combined_text = f"{pending_reminder} {text}"
                self.logger.info(f"Обработка времени: '{combined_text}'")

                # Снова отправляем в NLP сервис
                nlp_result = await self.nlp_client.parse_text(combined_text, user_id)

                if nlp_result and nlp_result.get("remind_at"):
                    return await self.reminder_handler.handle_reminder_creation(user_id, nlp_result)
                else:
                    # Если снова не удалось распознать время
                    return BotResponse(
                        text="❌ Не удалось распознать время. Попробуй еще раз, например: 'завтра в 15:00' или 'сегодня вечером'"
                    )
            else:
                # Контекст потерян, начинаем заново
                await self.state_manager.clear_state(user_id)
                return BotResponse(
                    text="❌ Что-то пошло не так. Давай начнем заново. О чем напомнить?"
                )

        except Exception as e:
            self.logger.error(f"Ошибка обработки ответа с временем: {e}")
            await self.state_manager.clear_state(user_id)
            return BotResponse(
                text="❌ Произошла ошибка. Давай начнем заново. О чем напомнить?"
            )

