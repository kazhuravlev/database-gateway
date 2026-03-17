package gateway

default allow_target := false
default allow_query := false

# Team ownership encoded in target ids:
# payments-prod, payments-staging, search-dev, etc.
allow_target if {
	"role:team-payments" in input.subjects
	startswith(input.target, "payments-")
}

allow_query if {
	"role:team-payments" in input.subjects
	startswith(input.target, "payments-dev")
}

allow_query if {
	"role:team-payments" in input.subjects
	startswith(input.target, "payments-staging")
	input.op == "select"
}

allow_query if {
	"role:team-payments" in input.subjects
	startswith(input.target, "payments-prod")
	input.op == "select"
}

