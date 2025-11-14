from datetime import datetime, timedelta
from typing import Any, Dict
import pytz


class RuleEngine:
    def validate_and_correct(self, data: Dict[str,Any]) -> Dict[str,Any]:
        corrected = data.copy()

        if not corrected.get('text'):
            corrected['text'] = "Напоминание"

        complexity = corrected.get('complexity', 2)
        if not isinstance(complexity, int) or complexity < 1 or complexity > 5:
            corrected['complexity'] = 2

        remind_at = corrected.get('remind_at')
        if remind_at:
            try:
                # Используем московское время для проверки
                moscow_tz = pytz.timezone('Europe/Moscow')

                dt_str = remind_at.replace('Z', '+00:00')
                parsed_date = datetime.fromisoformat(dt_str)

                if parsed_date.tzinfo is None:
                    parsed_date = moscow_tz.localize(parsed_date)
                else:
                    parsed_date = parsed_date.astimezone(moscow_tz)

                now = datetime.now(moscow_tz)
                if parsed_date < now:
                    parsed_date = parsed_date + timedelta(days=1)

                corrected['remind_at'] = parsed_date.strftime('%Y-%m-%dT%H:%M:%S%z')
                if len(corrected['remind_at']) == 25:
                    corrected['remind_at'] = corrected['remind_at'][:-2] + ':' + corrected['remind_at'][-2:]
            except (ValueError, AttributeError) as e:
                corrected['remind_at'] = None
        return corrected