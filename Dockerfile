FROM python:3-alpine
MAINTAINER Philipp Hellmich <phil@hellmi.de>

WORKDIR /usr/src/app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

USER nobody

CMD [ "python", "./exporter.py" ]

EXPOSE 8000
