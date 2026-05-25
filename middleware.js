const REALM = "SQL Playground Admin";

export const config = {
  matcher: ["/admin", "/admin/:path*", "/api/admin/:path*"]
};

export default function middleware(request) {
  const username = process.env.ADMIN_USERNAME;
  const password = process.env.ADMIN_PASSWORD;

  if (!username || !password) {
    return new Response("Admin credentials are not configured.", {
      status: 500,
      headers: {
        "content-type": "text/plain; charset=utf-8"
      }
    });
  }

  const authorization = request.headers.get("authorization") || "";
  if (isAuthorized(authorization, username, password)) {
    return;
  }

  return new Response("Authentication required.", {
    status: 401,
    headers: {
      "content-type": "text/plain; charset=utf-8",
      "www-authenticate": `Basic realm="${REALM}", charset="UTF-8"`
    }
  });
}

function isAuthorized(authorization, username, password) {
  if (!authorization.startsWith("Basic ")) {
    return false;
  }

  const encoded = authorization.slice("Basic ".length).trim();
  let decoded;
  try {
    decoded = atob(encoded);
  } catch {
    return false;
  }

  const separatorIndex = decoded.indexOf(":");
  if (separatorIndex < 0) {
    return false;
  }

  const actualUsername = decoded.slice(0, separatorIndex);
  const actualPassword = decoded.slice(separatorIndex + 1);
  return actualUsername === username && actualPassword === password;
}
