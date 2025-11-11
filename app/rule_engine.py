from datetime import datetime
from typing import Any, Dict


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
                datetime.fromisoformat(remind_at.replace('Z', '+00:00'))
            except ValueError:
                corrected['remind_at'] = None
        return corrected