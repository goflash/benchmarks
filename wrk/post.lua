wrk.method = "POST"
wrk.body   = '{"name":"flash","value":123,"ok":true,"items":[1,2,3]}'
wrk.headers["Content-Type"] = "application/json"

-- Optionally allow overriding the body via an env var for future extensions
local envBody = os.getenv("WRK_BODY")
if envBody and envBody ~= "" then
  wrk.body = envBody
end
