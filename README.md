About RNA
---

RNA is a simple (and incomplete) DNS Cache.

Should i use RNA?
---

Probably not. While it *is* somewhat useable and can be used for casual surfing, it is far far faaar away from being stable.

Some bug highlights:

* Does not validate any replies - DNS Cache poisoning ahoi!
* Fails to decompress any non NS/CNAME RR (you'll get funny dig output)
* The negative cache never expires
* Can only talk to IPv4 Nameservers
* No loop protection (eg: cnames pointing to each other, endless delegations, etc)

Why?!
---
Because i wanted do something fun in Go.

What does RNA mean?
---
'DNS' is the german abbreviation for DNA (Deoxyribonucleic acid -> Desoxyribonuklein*säure*) and RNA is not really a DNS. The name was supposed to be funny. somewhat.

