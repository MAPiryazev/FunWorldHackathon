from datetime import datetime
from typing import Dict

from app.gigachat_client import GigaChatClient
from app.models import ExtractedData, Priority, Category
from app.rule_engine import RuleEngine



class TextParser:
    def __init__(self):
        self.gigachat = GigaChatClient()  # Клиент для GigaChat
        self.rule_engine = RuleEngine()   # Валидатор данных


    async def parse_reminder_text(self, text: str) -> ExtractedData:

        # 1. Получаем данные от GigaChat
        gigachat_data = await self.gigachat.extract_reminder_data(text)

        # 2. Валидируем и корректируем
        validated_data = self.rule_engine.validate_and_correct(gigachat_data)

        # 3. Конвертируем в нашу модель
        return self._convert_to_model(validated_data)

    def _convert_to_model(self, data: Dict) -> ExtractedData:
        remind_at = None

        if data.get('remind_at'):
            try:
                dt_str = data['remind_at'].replace('Z', '+00:00')
                remind_at = datetime.fromisoformat(dt_str)
            except ValueError as e:
                print(f"Ошибка парсинга даты: {e}")

        return ExtractedData(
            text=data['text'],
            remind_at=remind_at,
            priority=Priority(data.get('priority', 'medium')),
            category=Category(data.get('category', 'другие')),
            complexity=data.get('complexity', 2),
            notes=data.get('notes')
        )