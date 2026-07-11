import base64, struct

# Build DNS query for example.com A record
q = bytearray()
q += b'\xaa\xaa'  # Transaction ID
q += b'\x01\x00'  # Flags: standard query
q += b'\x00\x01'  # Questions: 1
q += b'\x00\x00'  # Answer RRs: 0
q += b'\x00\x00'  # Authority RRs: 0
q += b'\x00\x00'  # Additional RRs: 0
# Question: example.com
q += b'\x07example\x03com\x00'  # Name
q += b'\x00\x01'  # Type: A
q += b'\x00\x01'  # Class: IN

open('/tmp/dns_query.bin', 'wb').write(q)
dns_b64 = base64.urlsafe_b64encode(q).rstrip(b'=').decode()
print(f'base64url: {dns_b64}')

# Test GET method
import urllib.request, ssl
ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE

url = f'https://doh.lllllyccc.qzz.io/dns-query?dns={dns_b64}'
req = urllib.request.Request(url, headers={'Accept': 'application/dns-message'})
resp = urllib.request.urlopen(req, context=ctx)
data = resp.read()
print(f'GET response: {len(data)} bytes')

# Parse A record from response
if len(data) > 12:
    flags = struct.unpack('!H', data[2:4])[0]
    rcode = flags & 0x000F
    ancount = struct.unpack('!H', data[6:8])[0]
    print(f'Rcode: {rcode}, Answers: {ancount}')
    if ancount > 0:
        offset = 12
        while data[offset] != 0:
            offset += data[offset] + 1
        offset += 5
        for i in range(ancount):
            if data[offset] & 0xC0 == 0xC0:
                offset += 2
            else:
                while data[offset] != 0:
                    offset += data[offset] + 1
                offset += 1
            atype = struct.unpack('!H', data[offset:offset+2])[0]
            rdlen = struct.unpack('!H', data[offset+8:offset+10])[0]
            if atype == 1:
                ip = '.'.join(str(b) for b in data[offset+10:offset+10+rdlen])
                print(f'Resolved: example.com -> {ip}')
            offset += 10 + rdlen

# Test POST method
req2 = urllib.request.Request(url.split('?')[0], data=q, method='POST')
req2.add_header('Content-Type', 'application/dns-message')
req2.add_header('Accept', 'application/dns-message')
resp2 = urllib.request.urlopen(req2, context=ctx)
data2 = resp2.read()
print(f'POST response: {len(data2)} bytes, same as GET: {data == data2}')
