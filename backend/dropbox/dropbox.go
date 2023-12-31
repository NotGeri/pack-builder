package dropbox

/*
https://www.dropbox.com/oauth2/authorize?client_id=xxx&response_type=code&redirect_uri=http://localhost/
http://localhost/?code=xxx
POST https://api.dropboxapi.com/oauth2/token (37a31da64d9bbd6c1a237b10a769c7.png)

Create a folder: POST https://api.dropboxapi.com/2/files/create_folder_v2 Body: {"autorename":false,"path":"/Homework/math"}
Upload <150 MB file: POST https://content.dropboxapi.com/2/files/upload Header: Dropbox-API-Arg={"autorename":false,"mode":"add","mute":false,"path":"/Homework/math/Matrices.txt","strict_conflict":false} body is an application/octet-stream
*/
