NGINX base image

### Tests for the patched nginx

`tests/` runs nginx's own test suite, [nginx-tests](https://github.com/nginx/nginx-tests), against nginx built from the source and patch stack in `rootfs/`.
The nginx version, its sha256, the patch loop and the core configure flags are read from `rootfs/build.sh`, so the tests follow the base image.
The test binary leaves out the third-party modules and is built with AddressSanitizer, so a test file also fails when the sanitizer reports a memory error.
CI runs the suite on every pull request that touches `images/nginx/`.

Run it locally with Docker:

```console
make -C images/nginx test                                    # whole suite, patched nginx
make -C images/nginx test TESTS="proxy.t t/harness.t"        # selected files, verbose
make -C images/nginx test NGINX_TESTS_FLAVOR=pristine TESTS=proxy.t    # stock nginx, no patches
```

`rootfs/patches/00_drop-alias-root.patch` removes the `root` and `alias` directives, but Test::Nginx puts `root` into every HTTP test configuration.
For a binary without `root`, `tests/nginx-tests-no-root.patch` makes Test::Nginx leave that line out and serve the test directory as the default root.
Upstream files whose own configuration uses `root` or `alias` are skipped on the patched build, and `tests/skip.txt` lists any other skipped files and why.
A file that fails is re-run once, verbosely, and the run fails only if it fails again.
Test::Nginx settings such as `TEST_NGINX_CATLOG=1` can be passed with `docker run -e` on the `nginx-tests:<TAG>-<flavor>` image.

To add a regression test for a backported fix, add a Test::Nginx script to `tests/t/` (`tests/t/harness.t` is a minimal example).
The pull request should show the test failing with `NGINX_TESTS_FLAVOR=pristine` and passing on the patched build.
The upstream suite is pinned by commit and sha256 in `tests/Dockerfile`; update both together.

### HTTP/3 Support

**HTTP/3 support is experimental and under development**

[HTTP/3](https://datatracker.ietf.org/doc/html/rfc9114)\
[QUIC](https://datatracker.ietf.org/doc/html/rfc9000)

[According to the documentation, NGINX 1.25.0 or higher supports HTTP/3:](https://nginx.org/en/docs/quic.html)

> Support for QUIC and HTTP/3 protocols is available since 1.25.0.

But this requires adding a new flag during the build:

> When configuring nginx, it is possible to enable QUIC and HTTP/3 using the --with-http_v3_module configuration parameter.

[We have added this flag](https://github.com/kubernetes/ingress-nginx/pull/11470), but it is not enough to use HTTP/3 in ingress-nginx, this is the first step.

The next steps will be:

1. **Waiting for OpenSSL 3.4.**\
    The main problem is, that we still use OpenSSL (3.x) and it does not support the important mechanism of TLS 1.3 - [early_data](https://datatracker.ietf.org/doc/html/rfc8446#section-2.3):

    > Otherwise, the OpenSSL compatibility layer will be used that does not support early data.
    
    [And although another part of the documentation says that the directive is supported with OpenSSL:](https://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_early_data)

    > The directive is supported when using OpenSSL 1.1.1 or higher.
    
    But this is incomplete support, because OpenSSL does not support this feature, and [it has only client side support:](https://github.com/openssl/openssl)

    > ... the QUIC (currently client side only) version 1 protocol
   
    [And also there are some issues even with client side](https://github.com/openssl/openssl/discussions/23339)

    Due to this, we currently have incomplete HTTP/3 support, without important security and performance features.\
    But the good news is that [OpenSSL plans to add server-side support in 3.4](https://github.com/openssl/web/blob/master/roadmap.md):

    > Server-side QUIC support
    
    [Overview of SSL libraries(HAProxy Documentation)](https://github.com/haproxy/wiki/wiki/SSL-Libraries-Support-Status#tldr)  

2. **Adding [parameters](https://nginx.org/en/docs/http/ngx_http_v3_module.html) to the configmap to configure HTTP/3 and quic(enableHTTP3, enableHTTP/0.9, maxCurrentStream, and so on).**
3. **Adding options to the nginx config template(`listen 443 quic` to server blocks and `add_header Alt-Svc 'h3=":8443"; ma=86400';` to location blocks).**
4. **Opening the https port for UDP in the container(because QUIC uses UDP).**
5. **Adding tests.**
