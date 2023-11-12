FROM python:3.11

COPY requirements.txt /requirements.txt
RUN python3 -m pip install -r /requirements.txt

COPY . /app
WORKDIR /app
CMD [ "python3", "main.py" ]
