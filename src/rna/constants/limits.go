package constants

const MAX_SIZE_LABEL int = 63           // maximum size of a label, excluding dot
const MAX_SIZE_NAME int = 255           // maximum size of a full dns name
const MAX_VALUE_TTL uint32 = 0xFFFFFFFF // maximum value of the TTL field
const MAX_SIZE_UDP int = 512            // max size of an incoming UDP query

const FIX_SIZE_HEADER int = 12 // header of a DNS query

// types as defined by RFC 1035 section 3.2.2
const (
	TYPE_NIL = iota
	TYPE_A
	TYPE_NS
	TYPE_MD
	TYPE_MF
	TYPE_CNAME
	TYPE_SOA
	TYPE_MB
	TYPE_MG
	TYPE_MR
	TYPE_NULL
	TYPE_WKS
	TYPE_PTR
	TYPE_HINFO
	TYPE_MINFO
	TYPE_MX
	TYPE_TXT

	TYPE_AAAA = 28

	QTYPE_AXFR  = 252
	QTYPE_MAILB = 253
	QTYPE_MAILA = 254
	QTYPE_ALL   = 255
)

// classes as defined by RFC 1035 Section 3.2.4
const (
	CLASS_NIL = iota
	CLASS_IN
	CLASS_CS
	CLASS_CH
	CLASS_HS
	QCLASS_ANY = 255
)

const (
	OP_QUERY  = 0
	OP_IQUERY = 1
	OP_STATUS = 2
)

const (
	RC_NO_ERR = iota
	RC_FORM_ERR
	RC_SERV_FAIL
	RC_NAME_ERR
	RC_NOT_IMPL
	RC_REFUSED
)
