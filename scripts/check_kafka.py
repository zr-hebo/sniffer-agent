import json
from kafka import KafkaConsumer, KafkaProducer



kafka_server = '192.168.XX.XX:9091'

group_id = 'sniffer'
topic = 'non_ddl_sql_collector'


def check_consume():
    conf = {
        'bootstrap_servers': kafka_server,
        'client_id': group_id,
        'group_id': group_id,
        'auto_offset_reset': 'earliest',
        'session_timeout_ms': 60000,
        'api_version': (0, 9, 0, 1)
    }
    consumer = KafkaConsumer(topic, **conf)
    print('ready to consume')
    for msg in consumer:
        event = json.loads(bytes.decode(msg.value))
        print(event)


def check_produce():
    conf = {
        'bootstrap_servers': kafka_server,
        'client_id': group_id
    }
    # 'api_version': (0, 9, 0, 1)
    producer = KafkaProducer(**conf)
    try:
        future = producer.send(topic, 'haha')
        result = future.get(timeout=3)
        print('send OK')
        print(result)

    except BaseException as e:
        print('send failed')
        # 发送失败时，用户需根据业务逻辑做异常处理，否则消息可能会丢失
        print(str(e))


def _real_main():
    # check_produce()
    check_consume()


if __name__ == '__main__':
    _real_main()
