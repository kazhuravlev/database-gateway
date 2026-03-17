package gateway

default allow_target := false
default allow_query := false

allow_target if {
	"role:user" in input.subjects
	input.target == "local-1"
}

allow_query if {
	"role:user" in input.subjects
	input.target == "local-1"
	input.op == "select"
	input.table == "public.clients"
}
