package gateway

default allow_target := false
default allow_vector := false

# Break-glass access is explicit and narrow.
allow_target if {
	"role:oncall" in input.subjects
	input.target == "taxi-prod"
}

allow_vector if {
	"role:oncall" in input.subjects
	input.target == "taxi-prod"
	input.op == "select"
}

allow_vector if {
	"user:sre-lead@example.com" in input.subjects
	input.target == "taxi-prod"
	input.op == "update"
	input.table == "public.drivers"
}

