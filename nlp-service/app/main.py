from contextlib import asynccontextmanager
import time
import logging
from fastapi import FastAPI
from app.models import ParseResponse, ParseRequest
from app.parser import TextParser


# Настройка логирования
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Глобальная переменная для экземпляра парсера
text_parser = None


@asynccontextmanager
async def lifespan(app: FastAPI):

    global text_parser
    try:
        logger.info("Инициализация NLP Service...")

        print("DEBUG: Создание экземпляря TextParser...")
        text_parser = TextParser()
        print(f"DEBUG: text_parser создан: {text_parser}")
        print(f"DEBUG: type: {type(text_parser)}")

        logger.info("NLP Service успешно запущен!")

    except Exception as e:
        logger.error(f"Ошибка запуска: {e}")
        raise

    yield

    # Shutdown
    logger.info("NLP Service остановлен")


app = FastAPI(
    title="NLP Service",
    description="Микросервис для парсинга напоминаний с GigaChat",
    lifespan=lifespan
)


@app.post("/parse", response_model=ParseResponse)
async def parse_text(request: ParseRequest):
    """Основной эндпоинт для парсинга текста"""
    print("DEBUG: /parse endpoint вызван!")
    start_time = time.time()

    try:
        logger.info(f"Получен запрос от пользователя {request.user_id}")
        logger.info(f"Текст для парсинга: {request.text}")

        print("DEBUG: Вызываем parser...")

        # Парсим текст через GigaChat
        extracted_data = await text_parser.parse_reminder_text(request.text)
        print(f"DEBUG: Parser вернул: {extracted_data}")

        logger.info(f"Парсинг успешен: {extracted_data.text}")

        return ParseResponse(
            user_id=request.user_id,
            text=extracted_data.text,
            remind_at=extracted_data.remind_at,
            complexity=extracted_data.complexity,
            priority=extracted_data.priority,
            category=extracted_data.category,
            notes=extracted_data.notes
        )

    except Exception as e:
        print(f"DEBUG: ОШИБКА В /parse: {e}")
        import traceback
        traceback.print_exc()

        return ParseResponse(
            user_id=request.user_id,
            text=request.text,
            remind_at=None,
            complexity=1,
            priority="medium",
            category="другие",
            notes= None
        )




if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000, reload=True)