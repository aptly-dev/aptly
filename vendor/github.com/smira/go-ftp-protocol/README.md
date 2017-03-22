go-ftp-protocol
===============

Plugin for http.Transport with support for ftp:// protocol in Go.

Limitations: only anonymous FTP servers, only file retrieval operations.

Internally connections to FTP servers are cached and re-used when possible.

Example usage:

    import "github.com/smira/go-ftp-protocol/protocol"

    transport := &http.Transport{}
    transport.RegisterProtocol("ftp", &protocol.FTPRoundTripper{})

    client := &http.Client{Transport: transport}

    resp, err := client.Get("ftp://ftp.ru.debian.org/debian/README")

License: MIT

Base on FTP client library: http://github.com/jlaffaye/ftp/