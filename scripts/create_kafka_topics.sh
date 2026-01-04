#!/bin/bash

# Создание Kafka топиков
KAFKA_BROKER=${KAFKA_BROKER:-localhost:9092}

echo "Creating Kafka topics..."

# Создаем топики
kafka-topics --bootstrap-server $KAFKA_BROKER --create --if-not-exists \
  --topic skin.price.updated \
  --partitions 3 \
  --replication-factor 1

kafka-topics --bootstrap-server $KAFKA_BROKER --create --if-not-exists \
  --topic skin.discovered \
  --partitions 3 \
  --replication-factor 1

kafka-topics --bootstrap-server $KAFKA_BROKER --create --if-not-exists \
  --topic notification.price_alert \
  --partitions 3 \
  --replication-factor 1

echo "Topics created successfully!"
kafka-topics --bootstrap-server $KAFKA_BROKER --list

