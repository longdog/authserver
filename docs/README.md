
### login

POST /auth/login

{
    "username":"admin",
    "password":"Aa12345"
}

->
AUTH_CODE


POST /auth/token

{
    "authCode":"AUTH_CODE"
}

->
{
    "refreshToken":"REFRESH_TOKEN",
    "authToken":"AUTH_TOKEN",
}

### logout

GET /auth/logout

### refresh

POST /auth/refresh

{
    "refreshToken":"REFRESH_TOKEN"
}

->
{
    "refreshToken":"REFRESH_TOKEN",
    "authToken":"AUTH_TOKEN",
}