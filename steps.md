 first of all i will create jwt because its the authentication layer and without it the api gateway would not be secure 
 **jwt validation**
1)Check token signature
2)expiry
3)reject invalid requests 

 Installing the Package 
go get github.com/golang-jwt/jwt/v5

jwt has 
---> header the algorithm which is HS256
---> payload the data
---->signature -->secret key 

the function in auth.go 

 createToken(username)---> 
 generates a jwt token that says who it belongs to (username)  and its expiration date and then combines the data with the secret key (in order to avoid tampering with the data)

 ValidateToken---> takes the jwt from client request aaand uses the secret key to verify if the data has been tampered with or not (validity check)