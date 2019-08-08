import time
import mysql.connector
from mysql.connector.cursor import MySQLCursorPrepared

config = {
    'host': '192.168.XX.XX',
    'port': 3358,
    'database': 'sniffer',
    'user': 'root',
    'password': '',
    'charset': 'utf8',
    'use_unicode': True,
    'get_warnings': True,
}


def _real_main(config):
    while True:
        _once_check()


def _once_check():
    output = []
    conn = mysql.connector.Connect(**config)

    curprep = conn.cursor(cursor_class=MySQLCursorPrepared)
    cur = conn.cursor()

    # Drop table if exists, and create it new
    stmt_drop = "DROP TABLE IF EXISTS names"
    cur.execute(stmt_drop)

    stmt_create = (
        "CREATE TABLE names ("
        "id TINYINT UNSIGNED NOT NULL AUTO_INCREMENT, "
        "name VARCHAR(30) DEFAULT '' NOT NULL, "
        "cnt TINYINT UNSIGNED DEFAULT 0, "
        "PRIMARY KEY (id))"
        )
    cur.execute(stmt_create)

    # Connector/Python also allows ? as placeholders for MySQL Prepared
    # statements.
    prepstmt = "INSERT INTO names (name) VALUES (%s)"

    # Preparing the statement is done only once. It can be done before
    # without data, or later with data.
    curprep.execute(prepstmt)

    # Insert 3 records
    names = (
        'Geert', 'Jan', 'Michel', 'wang', 'Jan', 'Michel', 'wang',
        'Jan', 'Michel', 'wang', 'Jan', 'Michel', 'wang', 'Jan', 'Michel',
        'wang', 'Jan', 'Michel', 'wang', 'Jan', 'Michel', 'wang', 'Jan',
        'Jan', 'Michel', 'wang')
    for name in names:
        curprep.execute(prepstmt, (name,))
        conn.commit()
        time.sleep(0.1)

    # We use a normal cursor issue a SELECT
    output.append("Inserted data")
    cur.execute("SELECT id, name FROM names")
    for row in cur:
        output.append("{0} | {1}".format(*row))

    # Cleaning up, dropping the table again
    cur.execute(stmt_drop)

    conn.close()
    print(output)


if __name__ == '__main__':
    _real_main(config)
