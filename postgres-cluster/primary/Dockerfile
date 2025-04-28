FROM postgres:15

COPY primary.conf /etc/postgresql/postgresql.conf
COPY pg_hba.conf /etc/postgresql/pg_hba.conf

CMD ["postgres", "-c", "config_file=/etc/postgresql/postgresql.conf", "-c", "hba_file=/etc/postgresql/pg_hba.conf"]