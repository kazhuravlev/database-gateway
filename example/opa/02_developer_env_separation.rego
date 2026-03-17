package gateway

default allow_target := false
default allow_query := false

# Developers can fully work in dev databases and read from staging.
allow_target if {
	"role:developer" in input.subjects
	startswith(input.target, "dev-")
}

allow_target if {
	"role:developer" in input.subjects
	startswith(input.target, "staging-")
}

allow_query if {
	"role:developer" in input.subjects
	startswith(input.target, "dev-")
}

allow_query if {
	"role:developer" in input.subjects
	startswith(input.target, "staging-")
	input.op == "select"
}

