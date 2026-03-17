package gateway

default allow_target := false
default allow_vector := false

# Analysts may inspect production data, but only with read-only access.
allow_target if {
	"role:analyst" in input.subjects
	startswith(input.target, "prod-")
}

allow_vector if {
	"role:analyst" in input.subjects
	startswith(input.target, "prod-")
	input.op == "select"
}

