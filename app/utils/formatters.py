from datetime import datetime
from typing import List, Dict, Any


def format_datetime(dt_string: str) -> str:
    """Форматирование даты-времени в читаемый вид"""
    try:
        dt = datetime.fromisoformat(dt_string.replace('Z', '+00:00'))
        return dt.strftime("%d.%m.%Y в %H:%M")
    except:
        return dt_string


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
            time_str = format_datetime(task.get('remind_at', ''))
            result += f"• {task['text']} - {time_str}\n"

    return result

def format_complexity(complexity: int) -> str:
    """Форматирование сложности в виде прогресс-бара"""
    filled = "⭐" * complexity
    empty = "☆" * (5 - complexity)
    return f"{filled}{empty} ({complexity}/5)"