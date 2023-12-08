"""
    Sidecar контейнер для логирования в MINIO
    Или любой другой object-storage с S3-compatible интерфейсом
"""
import os
import json
import yaml
from multiprocessing.pool import ThreadPool
from argparse import ArgumentParser, Namespace

from minio import Minio

from src.minio_worker import minio_worker
from src.utils.logging_utils import get_logger

MINIO_ENDPOINT = os.environ["S3_ENDPOINT"]
ACCESS_KEY = os.environ["S3_ACCESS_KEY_ID"]
SECRET_KEY = os.environ["S3_SECRET_ACCESS_KEY"]
S3_BUCKET = os.environ["S3_BUCKET"]

logger = get_logger(__name__)


def _get_args() -> Namespace:
    parser = ArgumentParser()
    parser.add_argument(
        "--streams_path",
        type=str,
        default="/config/streams.yaml",
        help="path to streams config file",
    )
    return parser.parse_args()


def s3log(
    streams_path: str,
):
    """
    Записываем логи в minio
    Args:
        streams_path: путь до файла с конфигом
    """
    with open(streams_path) as fin:
        streams = yaml.safe_load(fin)
    logger.info(json.dumps(streams, indent=4))
    # Запускаем поток, который будет бесконечно пушить файл в minio

    _client = Minio(
        MINIO_ENDPOINT,
        secure=False,
        access_key=ACCESS_KEY,
        secret_key=SECRET_KEY,
    )
    found = _client.bucket_exists(S3_BUCKET)
    if not found:
        logger.warning("Bucket %s not found; Creating one...", S3_BUCKET)
        _client.make_bucket(S3_BUCKET)

    with ThreadPool() as pool:
        for _ in pool.imap_unordered(minio_worker, streams):
            pass


if __name__ == "__main__":
    args = _get_args()
    s3log(
        streams_path=args.streams_path,
    )
