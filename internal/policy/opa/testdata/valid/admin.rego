package gateway

allow_target if {
	"role:admin" in input.subjects
}

allow_query if {
	"role:admin" in input.subjects
}
