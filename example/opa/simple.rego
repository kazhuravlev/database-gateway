package gateway

default allow_target := false
default allow_vector := false

allow_target if {
	"role:user" in input.subjects
	input.target == "local-1"
}

allow_vector if {
	"role:user" in input.subjects
	input.target == "local-1"
	input.op == "select"
	input.table == "public.clients"
}
