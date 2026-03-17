package gateway

default allow_target := false
default allow_query := false

allow_target if {
	"role:admin" in input.subjects
}

allowed_users_servers := {
	"taxi-prod",
	"pg-5435",
	"local-1",
	"local-2",
}

allow_target if {
	"role:user" in input.subjects
	input.target in allowed_users_servers
}

allow_query if {
	"role:admin" in input.subjects
}

allow_query if {
	"role:user" in input.subjects
	input.target == "local-1"
	input.op == "select"
}

allow_query if {
	"role:user" in input.subjects
	input.target == "local-2"
	input.op == "select"
}

allow_query if {
	"role:user" in input.subjects
	input.target in {"pg-5435", "taxi-prod"}
	input.op == "select"
	input.table in {"public.drivers", "public.clients", "public.transfers"}
}
