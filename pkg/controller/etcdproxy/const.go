package etcdproxy

// EtcdProxyServingCACert is CA certificate for aggregated API server to verify etcdproxy identity.
const EtcdProxyServingCACert = "-----BEGIN CERTIFICATE-----\n" +
	"MIICUjCCAfegAwIBAgIUEoYE1vzZ8qFZiwn1UcoLlDJfQ58wCgYIKoZIzj0EAwIw\n" +
	"ezELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1TYW4gRnJhbmNp\n" +
	"c2NvMUcwRQYDVQQDEz5ldGNkLXNhbXBsZS1hcGlzZXJ2ZXIua3ViZS1hcGlzZXJ2\n" +
	"ZXItc3RvcmFnZS5zdmMuY2x1c3Rlci5sb2NhbDAeFw0xODA3MDUxMTMzMDBaFw0y\n" +
	"MzA3MDQxMTMzMDBaMHsxCzAJBgNVBAYTAlVTMQswCQYDVQQIEwJDQTEWMBQGA1UE\n" +
	"BxMNU2FuIEZyYW5jaXNjbzFHMEUGA1UEAxM+ZXRjZC1zYW1wbGUtYXBpc2VydmVy\n" +
	"Lmt1YmUtYXBpc2VydmVyLXN0b3JhZ2Uuc3ZjLmNsdXN0ZXIubG9jYWwwWTATBgcq\n" +
	"hkjOPQIBBggqhkjOPQMBBwNCAASOJ2NjLdhNHRgyE66HWYv4KPT2QG4QM5JXaeN6\n" +
	"2TWVlfrefsr62Fu0preJTK7exhgjuDYk7dq8+AebnWU+Rujbo1kwVzAOBgNVHQ8B\n" +
	"Af8EBAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUat1Al6BEhcH57Jrw\n" +
	"bEH6Jc2aSFMwFQYDVR0RBA4wDIcEAAAAAIcEfwAAATAKBggqhkjOPQQDAgNJADBG\n" +
	"AiEA9fvh0DqhHlOBDBtQuUzotkX75dxyVLLVYEvRqSQPQ3cCIQDyImsQXneams3X\n" +
	"ffJ6NjNjtQracTYshEwLjZA4yDTTGA==\n" +
	"-----END CERTIFICATE-----"

// EtcdProxyClientCACert is CA certificate for etcdproxy to verify the aggregated API server.
// At this point they're same, but the controller by architecture allows different serving and client CA bundles.
const EtcdProxyClientCACert = EtcdProxyServingCACert

// EtcdProxyServerCert is etcdproxy server certificate.
const EtcdProxyServerCert = "-----BEGIN CERTIFICATE-----\n" +
	"MIIDFTCCArygAwIBAgIUflTgAFM4dzIiZFbcy5Xz1d3zbn8wCgYIKoZIzj0EAwIw\n" +
	"ezELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1TYW4gRnJhbmNp\n" +
	"c2NvMUcwRQYDVQQDEz5ldGNkLXNhbXBsZS1hcGlzZXJ2ZXIua3ViZS1hcGlzZXJ2\n" +
	"ZXItc3RvcmFnZS5zdmMuY2x1c3Rlci5sb2NhbDAeFw0xODA3MDUxMTM0MDBaFw0y\n" +
	"MzA3MDQxMTM0MDBaMHsxCzAJBgNVBAYTAlVTMQswCQYDVQQIEwJDQTEWMBQGA1UE\n" +
	"BxMNU2FuIEZyYW5jaXNjbzFHMEUGA1UEAxM+ZXRjZC1zYW1wbGUtYXBpc2VydmVy\n" +
	"Lmt1YmUtYXBpc2VydmVyLXN0b3JhZ2Uuc3ZjLmNsdXN0ZXIubG9jYWwwWTATBgcq\n" +
	"hkjOPQIBBggqhkjOPQMBBwNCAATg+vj0OXqB37mTn3EIxbK7STLfwz1NAMlg/Hzm\n" +
	"3i/E0I7ZWPXhL/RGzL/WMxmg/CJU+6KGjJRubcdRW1Nh0cjOo4IBHDCCARgwDgYD\n" +
	"VR0PAQH/BAQDAgWgMBMGA1UdJQQMMAoGCCsGAQUFBwMBMAwGA1UdEwEB/wQCMAAw\n" +
	"HQYDVR0OBBYEFO12z6buKWX+/cs3wzT+3BOaUYw1MIHDBgNVHREEgbswgbiCMGV0\n" +
	"Y2Qtc2FtcGxlLWFwaXNlcnZlci5rdWJlLWFwaXNlcnZlci1zdG9yYWdlLnN2Y4I4\n" +
	"ZXRjZC1zYW1wbGUtYXBpc2VydmVyLmt1YmUtYXBpc2VydmVyLXN0b3JhZ2Uuc3Zj\n" +
	"LmNsdXN0ZXKCPmV0Y2Qtc2FtcGxlLWFwaXNlcnZlci5rdWJlLWFwaXNlcnZlci1z\n" +
	"dG9yYWdlLnN2Yy5jbHVzdGVyLmxvY2FshwQAAAAAhwR/AAABMAoGCCqGSM49BAMC\n" +
	"A0cAMEQCID8YosXvr189gDahzp6iIbzzsQ7Hlzdnn+8uPMU0ulBNAiA74I9NqeOy\n" +
	"arH0bVBjF293EIwp32ezuwzDzNhDA7ptOA==\n" +
	"-----END CERTIFICATE-----"

// EtcdProxyServerKey is etcdproxy server key.
const EtcdProxyServerKey = "-----BEGIN EC PRIVATE KEY-----\n" +
	"MHcCAQEEIKcdeBjUQ3DobO453NG2MVbeZNJzG2vWXdtWeOlCSAZVoAoGCCqGSM49\n" +
	"AwEHoUQDQgAE4Pr49Dl6gd+5k59xCMWyu0ky38M9TQDJYPx85t4vxNCO2Vj14S/0\n" +
	"Rsy/1jMZoPwiVPuihoyUbm3HUVtTYdHIzg==\n" +
	"-----END EC PRIVATE KEY-----"

// EtcdProxyClientCert is etcdproxy client certificate.
const EtcdProxyClientCert = "-----BEGIN CERTIFICATE-----\n" +
	"MIICNTCCAdugAwIBAgIUJim7LWxnETjthrEMqyy9v3wo8D8wCgYIKoZIzj0EAwIw\n" +
	"ezELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1TYW4gRnJhbmNp\n" +
	"c2NvMUcwRQYDVQQDEz5ldGNkLXNhbXBsZS1hcGlzZXJ2ZXIua3ViZS1hcGlzZXJ2\n" +
	"ZXItc3RvcmFnZS5zdmMuY2x1c3Rlci5sb2NhbDAeFw0xODA3MDUxMTM4MDBaFw0y\n" +
	"MzA3MDQxMTM4MDBaMEMxCzAJBgNVBAYTAlVTMQswCQYDVQQIEwJDQTEWMBQGA1UE\n" +
	"BxMNU2FuIEZyYW5jaXNjbzEPMA0GA1UEAxMGY2xpZW50MFkwEwYHKoZIzj0CAQYI\n" +
	"KoZIzj0DAQcDQgAEJNnYlPCQEhA6MWtdzNAXyg8xQjvkguZgSaGC3T/b7mqc4aLd\n" +
	"Nx42N5jMdNb/ZNL8hPN2bN67SlR3esXXrit926N1MHMwDgYDVR0PAQH/BAQDAgWg\n" +
	"MBMGA1UdJQQMMAoGCCsGAQUFBwMCMAwGA1UdEwEB/wQCMAAwHQYDVR0OBBYEFCog\n" +
	"068UcXtIijQgZHbxX4sjePrgMB8GA1UdIwQYMBaAFGrdQJegRIXB+eya8GxB+iXN\n" +
	"mkhTMAoGCCqGSM49BAMCA0gAMEUCIDu7hgIsLpIFuiXi5T1+Gi2/rOSzCWJekmhR\n" +
	"Qow5xlcHAiEAlOARKaXlTrIhJkvVh/dclckJbFPGxps4HBlN1HYuJ/Y=\n" +
	"-----END CERTIFICATE-----"

// EtcdProxyClientKey is etcdproxy client key.
const EtcdProxyClientKey = "-----BEGIN EC PRIVATE KEY-----\n" +
	"MHcCAQEEICeuPjI1Rgd+veKAdHZf/iKDxlxQLKv/hPawuja5RRutoAoGCCqGSM49\n" +
	"AwEHoUQDQgAEJNnYlPCQEhA6MWtdzNAXyg8xQjvkguZgSaGC3T/b7mqc4aLdNx42\n" +
	"N5jMdNb/ZNL8hPN2bN67SlR3esXXrit92w==\n" +
	"-----END EC PRIVATE KEY-----"
