
from datetime import datetime, timezone, timedelta
import pytz

import httpx
import uuid
import logging
import json
from typing import Optional, Dict, Any
from app.config import settings


class GigaChatClient:
    """Клиент для работы с GigaChat API"""

    def __init__(self):
        self.logger = logging.getLogger(__name__)
        self.auth_key = settings.GIGACHAT_AUTH_KEY
        self.scope = settings.GIGACHAT_SCOPE
        self.auth_url = settings.GIGACHAT_AUTH_URL
        self.api_url = settings.GIGACHAT_API_URL
        self._access_token: Optional[str] = None

    async def _get_access_token(self) -> Optional[str]:
        """Получает access token для аутентификации"""
        try:
            rq_uid = str(uuid.uuid4())

            async with httpx.AsyncClient(verify=False) as client:
                response = await client.post(
                    self.auth_url,
                    data={"scope": self.scope},
                    headers={
                        "Content-Type": "application/x-www-form-urlencoded",
                        "Accept": "application/json",
                        "RqUID": rq_uid,
                        "Authorization": f"Bearer {self.auth_key}"
                    },
                    timeout=30.0
                )

                if response.status_code == 200:
                    token_data = response.json()
                    access_token = token_data.get("access_token")
                    if access_token:
                        self.logger.info("Токен доступа успешно получен")
                        return access_token
                    else:
                        self.logger.error("Токен не найден в ответе")
                        return None
                else:
                    self.logger.error(f"Ошибка аутентификации: {response.status_code}")
                    self.logger.error(f"Тело ответа: {response.text}")
                    return None

        except Exception as e:
            self.logger.error(f"Ошибка при получении токена: {e}")
            return None

    async def _ensure_valid_token(self) -> Optional[str]:
        """Проверяет и обновляет токен при необходимости"""
        return await self._get_access_token()

    async def extract_reminder_data(self, text: str) -> Dict[str, Any]:
        """Извлекает структурированные данные из текста напоминания"""
        try:
            # Строит промпт для GigaChat
            prompt = self._build_prompt(text)

            # Отправляет запрос к GigaChat
            response_content = await self.send_message(prompt)

            # Парсит JSON из ответа
            return self._parse_response(response_content)

        except Exception as e:
            self.logger.error(f"Ошибка извлечения данных: {e}")
            raise


    def _build_prompt(self, text: str) -> str:
        """Строит промпт для анализа напоминания"""
        # Используем московское время (UTC+3)
        moscow_tz = pytz.timezone('Europe/Moscow')
        moscow_now = datetime.now(moscow_tz)
        current_date = moscow_now.strftime("%Y-%m-%d")
        current_datetime = moscow_now.strftime("%Y-%m-%d %H:%M:%S")

        return f"""Ты - ассистент для парсинга текстовых напоминаний. Проанализируй текст и определи параметры задачи.
    Текущая дата и время: {current_datetime} (Москва, UTC+3)
    Текущая дата: {current_date}

    Текст: "{text}"
    
    ПРАВИЛА АНАЛИЗА:

    ПРАВИЛА ДЛЯ ДАТ:
    - ВСЕ ВРЕМЯ УКАЗЫВАЙ В МОСКОВСКОМ ЧАСОВОМ ПОЯСЕ (UTC+3)
    - Используй ТЕКУЩИЙ год если в тексте не указан конкретный год
    - "Сегодня" = текущая дата ({current_date})
    - "Завтра" = текущая дата + 1 день
    - "Послезавтра" = текущая дата + 2 дня  
    - "В понедельник" = ближайший понедельник от текущей даты
    - ВАЖНО: Если указано время (например "00:02") и это время УЖЕ ПРОШЛО сегодня, используй ЗАВТРА с этим временем
    - Если время не указано, используй 09:00:00 по умолчанию
    - Если дату невозможно определить, используй null
    - ВАЖНО: Время в remind_at должно быть в формате ISO с указанием часового пояса +03:00 (Москва)

    Примеры:
    - "сегодня в 00:02" -> если сейчас уже прошло 00:02, то завтра в 00:02, иначе сегодня в 00:02
    - "завтра в 10:00" -> текущая_дата + 1 день, время 10:00
    - "в понедельник" -> ближайший понедельник, время 09:00
    - "25 декабря" -> текущий_год-12-25, время 09:00
    - "позвонить маме" -> null (дата не указана)

    ПРАВИЛА РАЗДЕЛЕНИЯ TEXT И NOTES:
    - TEXT: только глагол + объект("позвонить маме", "купить молоко")
    - NOTES: всё остальное - причины, уточнения, детали("спросить про день рождения", "обязательно безлактозное")

    ПРАВИЛА АНАЛИЗА:

    КАТЕГОРИИ (category):
    - "работа" → рабочие задачи, встречи, дедлайны, проекты
    - "личное" → семья, друзья, хобби, личные дела
    - "здоровье" → врач, лекарства, спорт, анализы
    - "покупки" → магазины, продукты, вещи, заказы
    - "другие" → всё остальное

    ПРИОРИТЕТ (priority):
    - "high" → срочные дела, дедлайны, важные встречи, неотложные задачи
    - "medium" → обычные задачи, плановые дела, стандартные напоминания  
    - "low" → не срочные дела, можно отложить, второстепенные задачи

    СЛОЖНОСТЬ (complexity) от 1 до 5:
    - 1 → простые действия (позвонить, купить, написать)
    - 2 → задачи с 1-2 шагами (сходить в магазин, оплатить счет)
    - 3 → задачи с планированием (организовать встречу, приготовить ужин)
    - 4 → сложные многошаговые задачи (подготовить отчет, спланировать поездку)
    - 5 → очень сложные задачи (организовать мероприятие, большой проект)

    ПРИМЕРЫ РАЗБОРА:
    - "срочно позвонить врачу записаться на прием" 
      → category: "здоровье", priority: "high", complexity: 2

    - "купить молоко и хлеб по дороге домой"
      → category: "покупки", priority: "low", complexity: 1

    - "подготовить отчет для начальника к пятнице"
      → category: "работа", priority: "high", complexity: 4

    - "встретиться с друзьями в субботу вечером"
      → category: "личное", priority: "medium", complexity: 1

    - "записаться на курсы английского на следующей неделе"
      → category: "другие", priority: "medium", complexity: 
      
    Верни ТОЛЬКО JSON в следующем формате:
    {{
        "text": "очищенный текст задачи",
        "remind_at": "дата и время в ISO формате с московским часовым поясом (YYYY-MM-DDTHH:MM:SS+03:00) или null",
        "priority": "low/medium/high",
        "category": "работа/личное/покупки/здоровье/другие", 
        "complexity": 1-5,
        "notes": "дополнительные детали, уточнения, причины"
    }}
    
    ВАЖНО: remind_at должен быть в формате с московским часовым поясом, например: "2025-11-13T22:30:00+03:00"
    """

    def _parse_response(self, response_content: str) -> Dict[str, Any]:
        """Парсит ответ от GigaChat и извлекает JSON"""
        try:
            # Ищет JSON в ответе
            json_start = response_content.find('{')
            json_end = response_content.rfind('}') + 1

            if json_start == -1:
                raise ValueError("JSON не найден в ответе")

            json_str = response_content[json_start:json_end]
            self.logger.info(f"Извлечен JSON: {json_str}")

            # Парсит JSON
            parsed_data = json.loads(json_str)
            self.logger.info("JSON успешно распарсен")

            return parsed_data

        except json.JSONDecodeError as e:
            self.logger.error(f"Ошибка парсинга JSON: {e}")
            raise ValueError(f"Неверный формат JSON в ответе: {e}")

    async def send_message(self,
                           message: str,
                           model: str = "GigaChat",
                           temperature: float = 0.1,
                           max_tokens: int = 1000) -> Optional[str]:
        """Отправляет сообщение в GigaChat API"""
        try:
            access_token = await self._ensure_valid_token()
            if not access_token:
                self.logger.error("Не удалось получить access token")
                return None

            self.logger.info(f"Отправка сообщения в GigaChat...")

            async with httpx.AsyncClient(verify=False) as client:
                response = await client.post(
                    self.api_url,
                    headers={
                        "Authorization": f"Bearer {access_token}",
                        "Content-Type": "application/json",
                        "Accept": "application/json"
                    },
                    json={
                        "model": model,
                        "messages": [{"role": "user", "content": message}],
                        "temperature": temperature,
                        "max_tokens": max_tokens
                    },
                    timeout=30.0
                )

                self.logger.info(f"Ответ API: {response.status_code}")

                if response.status_code == 200:
                    result = response.json()
                    content = result["choices"][0]["message"]["content"]
                    self.logger.info("API запрос успешен!")
                    return content
                else:
                    self.logger.error(f"Ошибка API: {response.status_code}")
                    self.logger.error(f"Тело ответа: {response.text}")
                    return None

        except Exception as e:
            self.logger.error(f"Ошибка при отправке сообщения: {e}")
            return None

