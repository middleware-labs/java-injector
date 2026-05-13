package discovery

import "testing"

func TestClassifyIntegration(t *testing.T) {
	tests := []struct {
		name string
		proc *Process
		want IntegrationType
	}{
		// --- Redis ---
		{
			name: "redis by port",
			proc: makeProc("some-wrapper", "some-wrapper", 6379),
			want: IntegrationRedis,
		},
		{
			name: "redis by exe name",
			proc: makeProc("redis-server", "redis-server /etc/redis.conf"),
			want: IntegrationRedis,
		},
		{
			name: "redis by both signals",
			proc: makeProc("redis-server", "redis-server /etc/redis.conf", 6379),
			want: IntegrationRedis,
		},
		{
			name: "redis-cli excluded",
			proc: makeProc("redis-cli", "redis-cli -h localhost", 6379),
			want: "",
		},
		{
			name: "valkey-server detected as redis",
			proc: makeProc("valkey-server", "valkey-server /etc/valkey.conf", 6379),
			want: IntegrationRedis,
		},

		// --- MySQL ---
		{
			name: "mysql by port",
			proc: makeProc("wrapper", "wrapper", 3306),
			want: IntegrationMySQL,
		},
		{
			name: "mysqld by exe",
			proc: makeProc("mysqld", "mysqld --datadir=/var/lib/mysql"),
			want: IntegrationMySQL,
		},
		{
			name: "mariadbd by exe",
			proc: makeProc("mariadbd", "mariadbd --user=mysql"),
			want: IntegrationMySQL,
		},
		{
			name: "mysqldump excluded",
			proc: makeProc("mysqldump", "mysqldump --all-databases"),
			want: "",
		},

		// --- PostgreSQL ---
		{
			name: "postgres by port",
			proc: makeProc("wrapper", "wrapper", 5432),
			want: IntegrationPostgreSQL,
		},
		{
			name: "postgres by exe",
			proc: makeProc("postgres", "postgres -D /var/lib/postgresql/data"),
			want: IntegrationPostgreSQL,
		},
		{
			name: "psql excluded",
			proc: makeProc("psql", "psql -U postgres"),
			want: "",
		},
		{
			name: "postgres walwriter excluded",
			proc: makeProc("postgres", "postgres: walwriter"),
			want: "",
		},
		{
			name: "postgres autovacuum launcher excluded",
			proc: makeProc("postgres", "postgres: autovacuum launcher"),
			want: "",
		},

		// --- MongoDB ---
		{
			name: "mongodb by port",
			proc: makeProc("wrapper", "wrapper", 27017),
			want: IntegrationMongoDB,
		},
		{
			name: "mongod by exe",
			proc: makeProc("mongod", "mongod --dbpath /data/db"),
			want: IntegrationMongoDB,
		},
		{
			name: "mongos by exe",
			proc: makeProc("mongos", "mongos --configdb rs/host:27019"),
			want: IntegrationMongoDB,
		},
		{
			name: "mongosh excluded",
			proc: makeProc("mongosh", "mongosh --host localhost"),
			want: "",
		},

		// --- Kafka ---
		{
			name: "kafka by port",
			proc: makeProc("java", "java -cp /opt/kafka/libs/*", 9092),
			want: IntegrationKafka,
		},
		{
			name: "kafka by cmdline",
			proc: makeProc("java", "java kafka.Kafka /opt/kafka/config/server.properties"),
			want: IntegrationKafka,
		},
		{
			name: "confluent kafka by cmdline",
			proc: makeProc("java", "java io.confluent.kafka.server.KafkaStart"),
			want: IntegrationKafka,
		},
		{
			name: "kafka-topics excluded",
			proc: makeProc("java", "java kafka-topics --list --bootstrap-server localhost:9092"),
			want: "",
		},

		// --- RabbitMQ ---
		{
			name: "rabbitmq by port",
			proc: makeProc("beam.smp", "beam.smp", 5672),
			want: IntegrationRabbitMQ,
		},
		{
			name: "rabbitmq by cmdline",
			proc: makeProc("beam.smp", "/usr/lib/erlang/erts/bin/beam.smp -K true -- -root /usr/lib/erlang rabbit"),
			want: IntegrationRabbitMQ,
		},
		{
			name: "rabbitmqctl excluded",
			proc: makeProc("rabbitmqctl", "rabbitmqctl status"),
			want: "",
		},

		// --- Elasticsearch ---
		{
			name: "elasticsearch by port",
			proc: makeProc("java", "java -Xms1g", 9200),
			want: IntegrationElasticsearch,
		},
		{
			name: "elasticsearch by cmdline",
			proc: makeProc("java", "java org.elasticsearch.bootstrap.Elasticsearch"),
			want: IntegrationElasticsearch,
		},
		{
			name: "opensearch by cmdline",
			proc: makeProc("java", "java -Dopensearch.path.home=/opt/opensearch"),
			want: IntegrationElasticsearch,
		},

		// --- Cassandra ---
		{
			name: "cassandra by port",
			proc: makeProc("java", "java -cp /opt/cassandra", 9042),
			want: IntegrationCassandra,
		},
		{
			name: "cassandra by cmdline",
			proc: makeProc("java", "java org.apache.cassandra.service.CassandraDaemon"),
			want: IntegrationCassandra,
		},
		{
			name: "nodetool excluded",
			proc: makeProc("nodetool", "nodetool status"),
			want: "",
		},

		// --- Memcached ---
		{
			name: "memcached by port",
			proc: makeProc("wrapper", "wrapper", 11211),
			want: IntegrationMemcached,
		},
		{
			name: "memcached by exe",
			proc: makeProc("memcached", "memcached -m 64 -p 11211"),
			want: IntegrationMemcached,
		},

		// --- ZooKeeper ---
		{
			name: "zookeeper by port",
			proc: makeProc("java", "java -cp /opt/zk", 2181),
			want: IntegrationZookeeper,
		},
		{
			name: "zookeeper by cmdline",
			proc: makeProc("java", "java org.apache.zookeeper.server.quorum.QuorumPeerMain"),
			want: IntegrationZookeeper,
		},

		// --- etcd ---
		{
			name: "etcd by port",
			proc: makeProc("wrapper", "wrapper", 2379),
			want: IntegrationEtcd,
		},
		{
			name: "etcd by exe",
			proc: makeProc("etcd", "etcd --data-dir /var/lib/etcd"),
			want: IntegrationEtcd,
		},

		// --- Consul (MatchProcessRequired — exe/cmdline is the signal) ---
		{
			name: "consul with port and exe",
			proc: makeProc("consul", "consul agent -server", 8500),
			want: IntegrationConsul,
		},
		{
			name: "consul port only — no match",
			proc: makeProc("wrapper", "wrapper", 8500),
			want: "",
		},
		{
			name: "consul exe only — matches",
			proc: makeProc("consul", "consul agent -server"),
			want: IntegrationConsul,
		},

		// --- Vault (MatchProcessRequired — exe/cmdline is the signal) ---
		{
			name: "vault with port and exe",
			proc: makeProc("vault", "vault server -config=/etc/vault.hcl", 8200),
			want: IntegrationVault,
		},
		{
			name: "vault port only — no match",
			proc: makeProc("wrapper", "wrapper", 8200),
			want: "",
		},
		{
			name: "vault exe only — matches",
			proc: makeProc("vault", "vault server -config=/etc/vault.hcl"),
			want: IntegrationVault,
		},

		// --- CouchDB ---
		{
			name: "couchdb by port",
			proc: makeProc("beam.smp", "beam.smp -K true couchdb", 5984),
			want: IntegrationCouchDB,
		},
		{
			name: "couchdb by cmdline",
			proc: makeProc("beam.smp", "/usr/lib/erlang/erts/bin/beam.smp -- couchdb"),
			want: IntegrationCouchDB,
		},

		// --- ClickHouse (MatchProcessRequired — exe name is the signal) ---
		{
			name: "clickhouse with port and exe",
			proc: makeProc("clickhouse-server", "clickhouse-server --config=/etc/clickhouse-server/config.xml", 8123),
			want: IntegrationClickHouse,
		},
		{
			name: "clickhouse port only — no match",
			proc: makeProc("wrapper", "wrapper", 8123),
			want: "",
		},
		{
			name: "clickhouse exe only — matches",
			proc: makeProc("clickhouse-server", "clickhouse-server"),
			want: IntegrationClickHouse,
		},
		{
			name: "clickhouse-client excluded",
			proc: makeProc("clickhouse-client", "clickhouse-client --host localhost", 8123),
			want: "",
		},

		// --- InfluxDB (MatchProcessRequired — exe name is the signal) ---
		{
			name: "influxdb with port and exe",
			proc: makeProc("influxd", "influxd run", 8086),
			want: IntegrationInfluxDB,
		},
		{
			name: "influxdb port only — no match",
			proc: makeProc("wrapper", "wrapper", 8086),
			want: "",
		},
		{
			name: "influxdb exe only — matches",
			proc: makeProc("influxd", "influxd run"),
			want: IntegrationInfluxDB,
		},

		// --- NATS ---
		{
			name: "nats by port",
			proc: makeProc("wrapper", "wrapper", 4222),
			want: IntegrationNATS,
		},
		{
			name: "nats by exe",
			proc: makeProc("nats-server", "nats-server -m 8222"),
			want: IntegrationNATS,
		},

		// --- Nginx (MatchProcessRequired — exe name is the signal) ---
		{
			name: "nginx with port 80",
			proc: makeProc("nginx", "nginx -g daemon off;", 80),
			want: IntegrationNginx,
		},
		{
			name: "nginx without port — still matches by exe",
			proc: makeProc("nginx", "nginx -g daemon off;"),
			want: IntegrationNginx,
		},
		{
			name: "nginx on non-standard port",
			proc: makeProc("nginx", "nginx -g daemon off;", 8443),
			want: IntegrationNginx,
		},
		{
			name: "port 80 without nginx exe — no match",
			proc: makeProc("my-app", "my-app serve", 80),
			want: "",
		},

		// --- Apache (MatchProcessRequired) ---
		{
			name: "apache httpd with port 80",
			proc: makeProc("httpd", "httpd -DFOREGROUND", 80),
			want: IntegrationApache,
		},
		{
			name: "apache2 on non-standard port",
			proc: makeProc("apache2", "apache2 -k start", 81),
			want: IntegrationApache,
		},
		{
			name: "apache2 without port — still matches by exe",
			proc: makeProc("apache2", "apache2 -k start"),
			want: IntegrationApache,
		},

		// --- HAProxy (MatchProcessRequired) ---
		{
			name: "haproxy with port 443",
			proc: makeProc("haproxy", "haproxy -f /etc/haproxy/haproxy.cfg", 443),
			want: IntegrationHAProxy,
		},
		{
			name: "haproxy without port — still matches by exe",
			proc: makeProc("haproxy", "haproxy -f /etc/haproxy/haproxy.cfg"),
			want: IntegrationHAProxy,
		},

		// --- Global exclusions ---
		{
			name: "docker-proxy excluded even on known port",
			proc: makeProc("docker-proxy", "docker-proxy -proto tcp -host-ip 0.0.0.0 -host-port 5432", 5432),
			want: "",
		},
		{
			name: "docker-proxy excluded on redis port",
			proc: makeProc("docker-proxy", "docker-proxy -proto tcp -host-port 6379", 6379),
			want: "",
		},

		// --- Edge cases ---
		{
			name: "unknown process, no match",
			proc: makeProc("my-custom-app", "my-custom-app serve", 9999),
			want: "",
		},
		{
			name: "empty process, no match",
			proc: makeProc("", ""),
			want: "",
		},
		{
			name: "beam.smp with rabbit matches rabbitmq not couchdb",
			proc: makeProc("beam.smp", "beam.smp rabbit@myhost"),
			want: IntegrationRabbitMQ,
		},
		{
			name: "beam.smp without rabbit or couchdb — no match",
			proc: makeProc("beam.smp", "beam.smp"),
			want: "",
		},
		{
			name: "case insensitive exe match",
			proc: makeProc("Redis-Server", "Redis-Server /etc/redis.conf"),
			want: IntegrationRedis,
		},
		{
			name: "case insensitive cmdline match",
			proc: makeProc("java", "java KAFKA.Kafka /config/server.properties"),
			want: IntegrationKafka,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyIntegration(tt.proc)
			if got != tt.want {
				t.Errorf("ClassifyIntegration() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyIntegrations_Batch(t *testing.T) {
	procs := []*Process{
		makeProc("redis-server", "redis-server", 6379),
		makeProc("my-app", "my-app serve", 9999),
		makeProc("nginx", "nginx -g daemon off;", 80),
	}

	classifyIntegrations(procs)

	if got := procs[0].IntegrationType(); got != string(IntegrationRedis) {
		t.Errorf("procs[0] IntegrationType() = %q, want %q", got, IntegrationRedis)
	}
	if got := procs[1].IntegrationType(); got != "" {
		t.Errorf("procs[1] IntegrationType() = %q, want empty", got)
	}
	if got := procs[2].IntegrationType(); got != string(IntegrationNginx) {
		t.Errorf("procs[2] IntegrationType() = %q, want %q", got, IntegrationNginx)
	}
}

// makeProc builds a minimal Process for testing classification.
func makeProc(exeName, cmdLine string, ports ...uint16) *Process {
	details := make(map[string]any)
	if len(ports) > 0 {
		listeners := make([]Listener, len(ports))
		for i, p := range ports {
			listeners[i] = Listener{Protocol: "tcp", Address: "0.0.0.0", Port: p}
		}
		details[DetailListeners] = listeners
	}
	return &Process{
		ExecutableName: exeName,
		CommandLine:    cmdLine,
		Details:        details,
	}
}
