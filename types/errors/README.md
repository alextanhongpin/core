# Error Mapping to HTTP Status Code

|HTTP Status Code         |Domain Error Mapping|Explanation                    |
|-------------------------|--------------------|-------------------------------|
|400 Bad Request          |bad_input           |For unmarshal json error, or validation errors|
|401 Unauthorized         |unauthorized        |When user is not authenticated.|
|403 Forbidden            |forbidden           |When user is authenticated, but not authorized to perform the action or access the resource.|
|404 Not found            |not_found           |When a resource is not found.  |
|409 Conflict             |conflict, already_exists|When the request conflicts with the state of the server, e.g. status has been modified.|
|401 Gone                 |-                   |Used for short-lived entity, e.g. campaign, promotions. Otherwise, use 404.|
|412 Precondition Failed  |-                   |This is specifically for headers, use 422 instead.|
|422 Unprocessable Entity |unprocessable       |The request is valid, but cannot be processed due to failure to fulfil certain conditions.|
|500 Internal Server Error|internal, unknown   |Any errors not handled should return unknown. They should eventually be mapped to one of those errors above.|
