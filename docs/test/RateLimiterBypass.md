<img width="1819" height="537" alt="image" src="https://github.com/user-attachments/assets/8121777b-6b9f-4d19-ac8a-fbf8284681c9" />


Source: [Raxomara](https://medium.com/@raxomara/bypassing-rate-limits-all-known-techniques-25891bb5ca59)

# Bypassing Rate Limits: All Known Techniques

## Introduction
Rate limiting is a crucial security feature to prevent abuse by controlling the number of requests a user or system can make to a server within a certain timeframe. However, attackers have developed numerous techniques to bypass these limits and exploit the system.

From this article, I listed some of most popular use cases that you can use to bypass Rate Limiters.

<img width="1179" height="638" alt="image" src="https://github.com/user-attachments/assets/b64137c0-e7fa-4c60-952c-233003b24372" />

### 1. Adding Custom Headers (X-Forwarded-For, X-Real-IP)
Attackers can manipulate headers like X-Forwarded-For to trick the server into thinking requests come from different IPs.  
You can find a complete list of these headers from `/test/data`   
<div align="center">
  <img width="613" height="174" alt="image" src="https://github.com/user-attachments/assets/1abc40f2-9ad2-481d-9262-dd6a8b0b9a29" />
</div>

### 2. Changing User-Agent
You can use some real lists of user-agent strings to bypass, I recommend to use real one because user-agent strings.   
You can view it [here](https://deviceatlas.com/blog/list-of-user-agent-strings) to use.

> [!TIP]
> Only trust IPs from your reverse proxy.  
> Ignore all client supplied headers.  
> Use a fixed header like CF Connecting IP or the real remote address.  
> Normalize the IP before counting.

### 3. Using Capital Letters or URL Case Changes
Imagine you have a login page at the URL /login. The system enforces a rate limit to prevent brute force attacks by restricting the number of login attempts a user can make in a given time frame. For instance, after 5 failed attempts within a minute, further login attempts from the same IP or account may be blocked temporarily.

However, in some systems, the rate-limiting mechanism may rely on the exact string of the URL, including its casing. If the system does not normalize the case of the URL when applying the rate limit, it could treat /login and /logiN (with the capital “N”) as two distinct endpoints.
<div align="center">
  <img width="641" height="418" alt="image" src="https://github.com/user-attachments/assets/a517343a-a769-49a0-aa1e-f12fb62bc58a" />
</div>

### 4. Parameter Manipulation
Adding unnecessary parameters to URLs (e.g., `&random=value`) can make the requests appear unique, bypassing rate limits.

Example: `http://example.com/resource?id=123&random=987`

### 5. HTTP Method Switch (GET/POST/PUT/PATCH/HEAD/OPTIONS)
In certain situations, bypassing weakly configured [WAFs](https://www.cloudflare.com/learning/ddos/glossary/web-application-firewall-waf/) or blacklists can be achieved by switching the HTTP request method. This involves transferring `POST` parameters to the URL as `GET` parameters, or the reverse. Additionally, experimenting with methods like `PUT` or `PATCH` may yield similar results.

### 6. Protocol Downgrading (HTTP/1.1, HTTP/2)
Some applications apply rate-limiting logic differently for `HTTP/1.1` and `HTTP/2`, creating opportunities for attackers to exploit inconsistencies.
- **Track rate limits per HTTP connection**: This is common with `HTTP/1.1` because each request uses a separate connection, and rate limits may be enforced by counting the number of open connections.
- **Track rate limits per request in HTTP/2**: Since `HTTP/2` multiplexes requests, a rate limit may focus on the total number of requests made through the connection, not on the connection itself.
**Bypassing Rate Limits with Protocol Downgrading**
- Scenario 1: Bypassing `HTTP/2` rate limits by downgrading to `HTTP/1.1`, where rate limits may be less strict.
- Scenario 2: Exploiting `HTTP/2` multiplexing to send multiple requests in parallel over a single connection, bypassing connection-based rate limits.

### 7. The Route Alteration Strategy
Rate-limiting is often tied to particular endpoint routes, and altering the route slightly can sometimes bypass these limitations. By appending `/.` or `/..` to the endpoint, the request might no longer match the rate-limiting rules, allowing for additional requests.
How it works:
- Modification Technique: Append `/.` or `/..` to the route.
- Normalization Confusion: The server may strip these changes during normalization. As a result, both `/path` and `/path/..` could be treated as identical after the server processes them.
<div align="center">
  <img width="708" height="248" alt="image" src="https://github.com/user-attachments/assets/72bc52b1-fc16-4930-94b1-2a9a4dec4c92" />
</div>

### 8. Adding characters such as null bytes, carriage returns, or spaces.
In username-based rate limiting systems, it may be feasible to create inconsistencies between the code that processes login attempts and the code that enforces rate limiting by adding specific characters. For instance, appending characters like `%00` to an email, such as `example@email.com%00`, could result in the login code interpreting it as `example@email.com`, while the rate-limiting logic might see it as a separate entry, allowing one to bypass the rate limit.

Experiment with adding or prepending characters such as `%00`, `%20`, `%09`, `%0d`, or `%0a` for similar effects.
<div align="center">
  <img width="619" height="245" alt="image" src="https://github.com/user-attachments/assets/6c845d61-4d1b-4c8c-a0a1-b95e7a6013b2" />
</div>

### 9. Encoding
For instance, submitting `e%78ample@em%61il.com` may be interpreted as `example@email.com` by the authentication code, while the rate-limiting system could treat it as a distinct email address, thus bypassing rate limits.
<div align="center">
  <img width="610" height="248" alt="image" src="https://github.com/user-attachments/assets/f127c4b7-1977-4e46-9a7f-c8c36a421ce4" />
</div>

### 10. Bypassing CDN-based Rate Limiting
Attackers can access the origin server directly, bypassing CDN protections like Cloudflare by targeting the server’s IP address.

Example: `http://origin-ip-address/resource`

 ### 11. IP Rotation

This technique involves rotating IP addresses to bypass rate limits that track requests based on IP addresses. Attackers can use VPNs, proxies, or TOR to change their IP after reaching the limit.

### 12. Referer Header Spoofing

In very rare cases, Rate Limit may be implemented on the Referer header.

Attackers can spoof the Referer header to bypass rate limits that rely on this header to verify the source of the requests.

### 13. Cookie Manipulation

Clearing or changing cookies may reset the rate-limiting counter, tricking the server into treating the request as new.

