from datetime import datetime
from typing import List, Dict, Any
import pytz


def format_datetime(dt_string: str) -> str:
    """Форматирование даты-времени в читаемый вид (в московском времени)"""
    try:
        moscow_tz = pytz.timezone('Europe/Moscow')
        
        if isinstance(dt_string, str):
            dt_string = dt_string.replace('Z', '+00:00')
            dt = datetime.fromisoformat(dt_string)

            if dt.tzinfo is None:
                dt = pytz.UTC.localize(dt)

            dt_moscow = dt.astimezone(moscow_tz)
            return dt_moscow.strftime("%d.%m.%Y в %H:%M")
        else:
            if isinstance(dt_string, datetime):
                if dt_string.tzinfo is None:
                    dt_string = pytz.UTC.localize(dt_string)
                dt_moscow = dt_string.astimezone(moscow_tz)
                return dt_moscow.strftime("%d.%m.%Y в %H:%M")
            return str(dt_string)
    except Exception as e:
        import logging
        logging.getLogger(__name__).warning(f"Ошибка форматирования даты '{dt_string}': {e}")
        return str(dt_string)


def format_priority(priority: str) -> str:
    """Форматирование приоритета с эмодзи"""
    emojis = {
        "low": "🔵 Низкий",
        "medium": "🟡 Средний",
        "high": "🔴 Высокий"
    }
    return emojis.get(priority, "⚪ Неизвестно")


def format_category(category: str) -> str:
    """Форматирование категории с эмодзи"""
    emojis = {
        "работа": "💼 Работа",
        "личное": "👤 Личное",
        "здоровье": "🏥 Здоровье",
        "покупки": "🛒 Покупки",
        "другие": "📌 Другие"
    }
    return emojis.get(category, "📌 Другие")



def format_reminder_list(reminders: List[Dict[str, Any]]) -> str:
    """Форматирование списка напоминаний"""
    if not reminders:
        return "📭 У вас нет напоминаний"

    result = "📋 **Ваши напоминания:**\n\n"

    for i, reminder in enumerate(reminders, 1):
        time_str = format_datetime(reminder.get('remind_at', ''))
        priority_str = format_priority(reminder.get('priority', 'medium'))

        result += f"{i}. **{reminder['text']}**\n"
        result += f"   ⏰ {time_str}\n"
        result += f"   {priority_str}\n"

        if reminder.get('notes'):
            result += f"   📋 {reminder['notes']}\n"

        result += "\n"

    return result


def format_user_tasks(tasks: List[Dict[str, Any]]) -> str:
    """Форматирование списка задач пользователя"""
    if not tasks:
        return "📭 У вас нет активных задач"

    pending_tasks = [t for t in tasks if t.get('status') in ['pending', 'ready']]
    completed_tasks = [t for t in tasks if t.get('status') == 'sent']

    result = "📊 **Статистика задач:**\n\n"
    result += f"⏳ Активные: {len(pending_tasks)}\n"
    result += f"✅ Выполненные: {len(completed_tasks)}\n\n"

    if pending_tasks:
        result += "**Ближайшие напоминания:**\n"
        for task in pending_tasks[:5]:  # Показываем только 5 ближайших
            remind_at = task.get('remind_at', '')
            time_str = format_datetime(remind_at) if remind_at else "время не указано"
            task_text = task.get('text', 'Без названия')
            result += f"• {task_text} - {time_str}\n"

    return result

def format_complexity(complexity: int) -> str:
    """Форматирование сложности в виде прогресс-бара"""
    filled = "⭐" * complexity
    empty = "☆" * (5 - complexity)
    return f"{filled}{empty} ({complexity}/5)"