package gateway

default allow_target := false
default allow_vector := false

deny_prod if {
	"role:contractor" in input.subjects
	startswith(input.target, "prod-")
}

allow_target if {
	"role:contractor" in input.subjects
	startswith(input.target, "staging-")
}

allow_target if {
	"user:trusted.contractor@example.com" in input.subjects
	input.target == "prod-reporting"
}

allow_vector if {
	"role:contractor" in input.subjects
	startswith(input.target, "staging-")
	input.op == "select"
}

allow_vector if {
	"user:trusted.contractor@example.com" in input.subjects
	input.target == "prod-reporting"
	input.op == "select"
}

allow_target := false if {
	deny_prod
	not ("user:trusted.contractor@example.com" in input.subjects)
}

allow_vector := false if {
	deny_prod
	not ("user:trusted.contractor@example.com" in input.subjects)
}

