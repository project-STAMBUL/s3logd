import io
import os
import time
from typing import Dict
from datetime import datetime

from minio import Minio

from src.utils.logging_utils import get_logger

MINIO_ENDPOINT = os.environ["S3_ENDPOINT"]
ACCESS_KEY = os.environ["S3_ACCESS_KEY_ID"]
SECRET_KEY = os.environ["S3_SECRET_ACCESS_KEY"]
S3_BUCKET = os.environ["S3_BUCKET"]


def minio_worker(
    stream: Dict,
):
    """
    Пушим в MINIO
    Args:
        stream: Аргументы потока
                file - путь до файла
                pushRate - как часто пушим
    """
    file_path = stream["file"]
    push_rate = stream["pushRate"]
    logger = get_logger(f"Stream: {file_path}")

    _client = Minio(
        MINIO_ENDPOINT,
        secure=False,
        access_key=ACCESS_KEY,
        secret_key=SECRET_KEY,
    )

    while True:
        time.sleep(push_rate)
        if not os.path.exists(file_path):
            logger.warning("stream file %s doesn't exists", file_path)
            continue

        try:
            # Generating object name
            object_name = os.path.basename(file_path)
            # object_name = os.path.join(
            #     datetime.now().date().strftime("%Y-%m-%d"),
            #     "_".join([datetime.now().strftime("%H"), object_name]),
            # )
            with open(file_path, "rb") as fin:
                file_bytes = fin.read()
                _client.put_object(
                    S3_BUCKET,
                    object_name,
                    io.BytesIO(file_bytes),
                    length=len(file_bytes),
                )
            logger.info("Pushed %s to %s:%s", file_path, S3_BUCKET, object_name)
        except Exception as ex:
            logger.error("Error: %s", ex)
