/* 
	Digital Inventory web application 
*/
package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
)

type Role string

//Definition of the user roles
const (
	Administrator  Role = "Administrator"
	NormalEmployee Role = "Normal Employee"
)

//Attributes of a User include username, password and a position
type User struct {
	Username string
	Password string
	Position string
}

//Product is defined by its id, barcode, price, quantity and type
type Product struct {
	Id       string
	Barcode  string
	Price    float32
	Quantity int
	Type     string
}

var tpl *template.Template //tpl is the template  
var db *sql.DB //db is the database used in the program
var user User //user is the current logged in user

var store = sessions.NewCookieStore([]byte("super-secret"))//cookie store for generating cookies as a user logs in

//this is the main function where we make our connection with the database, include our templates and handlers and run the server
func main() {
	pswd := os.Getenv("DATABASE_PASS")
	var err error
	db, err = sql.Open("mysql", "root:"+pswd+"@tcp(localhost:3306)/goproject")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Println("error verifying connection with db.Ping")
		panic(err.Error())
	}

	tpl, err = template.ParseGlob("templates/*.html")
	if err != nil {
		fmt.Println("tell me there is an error")
	}
	http.HandleFunc("/login", processLoginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/", Auth(loginHandler))
	http.HandleFunc("/home", Auth(homeHandler))
	http.HandleFunc("/add", Auth(addHandler))
	http.HandleFunc("/delete", Auth(deleteHandler))
	http.HandleFunc("/check", Auth(checkHandler))
	http.HandleFunc("/show", Auth(showHandler))
	http.HandleFunc("/insert", Auth(insertHandler))
	http.HandleFunc("/remove", Auth(removeHandler))
	http.HandleFunc("/orders", Auth(ordersHandler))
	http.ListenAndServe("localhost:8080", context.ClearHandler(http.DefaultServeMux))
}

//in the Auth function we authenticate through cookies whether a user is logged in, so they can access the other functionalities 
//of the system
func Auth(HandlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		_, ok := session.Values["userID"]
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		HandlerFunc.ServeHTTP(w, r)
	}
}

//in the manageAccess function we check the role of the user, whether he/she is an admin or a regular employee
//as a regular employee does not have as many functionalities as an admin does
func manageAccess(w http.ResponseWriter, r *http.Request) {
	fmt.Println(user.Position)
	if user.Position == string(NormalEmployee){
		http.Redirect(w,r,"/home",http.StatusFound)
	}
}

//in loginHandler we execute the template for our log in page
func loginHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusFound)
	tpl.ExecuteTemplate(w, "template.html", nil)
}

//in processLoginHandler we check if the given username and password match with a user in our database table called 'users'
//if the username and password are correct, then the user is taken to the /home page and if not, they have to try with
//correct username and password in order to log in
func processLoginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("usernameName")
	password := r.FormValue("passwordData")

	stmt := "SELECT * FROM users WHERE username = ?;"
	row := db.QueryRow(stmt, username)
	err := row.Scan(&user.Username, &user.Password, &user.Position)
	if err != nil || user.Password != password {
		tpl.ExecuteTemplate(w, "template.html", nil)
		return
	}else{
			session, _ := store.Get(r, "session")
			session.Values["userID"] = user.Username
			session.Save(r, w)

			http.Redirect(w, r, "/home", http.StatusFound)
			return
		}

}

//in homeHandler if the logged in user is an admin, then a "homeAdmin.html" template is executed, which shows the
// additional functionalities if the user is a regular employee, then a "home.html" template is executed 
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if user.Position == string(NormalEmployee) {
		tpl.ExecuteTemplate(w, "home.html", user)
	}
	if user.Position == string(Administrator) {
		tpl.ExecuteTemplate(w, "homeAdmin.html", user)
	}
}

//in addHandler the user has the opportunity to increase the quantity of an already existing product in the database table clothes
//the addment happens via a valid id and a given number that shows the quantity
func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tpl.ExecuteTemplate(w, "add.html", nil)
		return
	}
	r.ParseForm()
	id := r.FormValue("idName")
	quantity := r.FormValue("quantityName")

	var err error

	idVar, err := strconv.Atoi(id)
	quantVar, err := strconv.Atoi(quantity)

	//check if quantity is negative
	if id == "" || quantity == "" {
		fmt.Println("everything is empty")
		tpl.ExecuteTemplate(w, "add.html", nil)
		return
	}

	var oldQuantity int

	stmt := "SELECT quantity FROM clothes WHERE id = ?;"
	row := db.QueryRow(stmt, idVar)
	err = row.Scan(&oldQuantity)

	var add *sql.Stmt
	add, err = db.Prepare("UPDATE `clothes` SET quantity = ? WHERE id = ?")
	if err != nil {
		//panic
		fmt.Println("error inserting")
	}
	defer add.Close()

	_, err = add.Exec(quantVar+oldQuantity, idVar)
	if err != nil {
		fmt.Println("error executing query")
		//panic
	}
	http.Redirect(w, r, "/home", http.StatusFound)
}

//in deleteHandler the user can decrease the quantity of an already existing product in the database table clothes, however
//the value of the quantity will not change in the case of an invalid number for quantity - this is whether the user
//wants to delete more pieces than there are in stock, or an invalid number is provided 
//if an invalid is provided, then the database will not be updated  
func deleteHandler(w http.ResponseWriter, r *http.Request) {//data validation  
	if r.Method == "GET" {
		tpl.ExecuteTemplate(w, "delete.html", nil)
		return
	}
	r.ParseForm()
	id := r.FormValue("idName")
	quantity := r.FormValue("quantityName")

	idVar, err := strconv.Atoi(id)
	quantVar, err := strconv.Atoi(quantity)

	if err!=nil{
		//panic
	}

	if id == "" || quantity == "" {
		fmt.Println("everything is empty")
		tpl.ExecuteTemplate(w, "delete.html", nil)
		return
	}

	flag,oldQuantity,message:=isQuantityEnough(quantVar,idVar)

	if flag==true {
		deleteProducts(oldQuantity,quantVar,idVar)
		tpl.ExecuteTemplate(w,"delete.html",message)
	}else{
		tpl.ExecuteTemplate(w, "delete.html", "You are trying to delete too many products")
		return
	}
}

//deleteProducts is a helper function which executes the decrease of quantity of a product by its id
func deleteProducts(oldQuantity int,quantVar int,idVar int){
	var delete *sql.Stmt
	delete, err := db.Prepare("UPDATE `clothes` SET quantity = ? WHERE id = ?")
	if err != nil {
		fmt.Println("error deleting")
	}
	defer delete.Close()

	_, err = delete.Exec(oldQuantity-quantVar, idVar)
	if err != nil {
		fmt.Println("error executing query")
	}
}

//checkHandler is a function which provides the functionality of checking the current quantity of a product by an id
//provided by the user and after checking in the database, shows a message in the /check handler about the current value
//of the product
func checkHandler(w http.ResponseWriter, r *http.Request) {//check if quantity is negative
	if r.Method == "GET" {
		tpl.ExecuteTemplate(w, "check.html", nil)
		return
	}
	r.ParseForm()
	id := r.FormValue("idName")

	idVar, err := strconv.Atoi(id)
	if err!=nil{
		fmt.Println("error converting id string to id int")
	}

	if id == ""  {
		fmt.Println("everything is empty")
		tpl.ExecuteTemplate(w, "check.html", nil)
		return
	}

	quantity:=getQuantityOfProduct(idVar)
	str:="The current quantity of product with id "+strconv.FormatInt(int64(idVar), 10)+" is "+strconv.FormatInt(int64(quantity), 10)
	tpl.ExecuteTemplate(w, "check.html", str)
}

//showHandler is a function which shows all ordered clothes in an order, specified by the user via its id 
//the function shows the id,barcode, price and type of the ordered products and the quantity ordered of any product
func showHandler(w http.ResponseWriter,r *http.Request){//authenticate idOrder
	if r.Method == "GET" {
		tpl.ExecuteTemplate(w, "show.html", nil)
		return
	}

	r.ParseForm()
	var idOrder string
	idOrder=r.FormValue("idName")
	//authenticate idOrder
	idOrderInt,_:=strconv.Atoi(idOrder)

	stmt, err := db.Prepare("SELECT idclothes,quantity FROM orderedclothes WHERE idorder = ?;")
	if err!=nil{
		fmt.Println("error with db.Prepare()")
	}
	defer stmt.Close()
	rows,err:=stmt.Query(idOrderInt)
	var products []Product

	for rows.Next() {
		var p Product
		var i int
		var orderedQuantity int
		
		err = rows.Scan(&i,&orderedQuantity)
		if err != nil {
			fmt.Println("unsuccessful scanning")
		}
		
		stmt:="SELECT * FROM clothes WHERE id = ?;"
		row:=db.QueryRow(stmt,i)
		err:=row.Scan(&p.Id,&p.Type,&p.Barcode,&p.Price,&p.Quantity)
		if err!=nil{
			fmt.Println("unsuccessful scanning")
		}
		p.Quantity=orderedQuantity
		products = append(products, p)
	}
	tpl.ExecuteTemplate(w, "show.html", products)
}

//getQuantityOfProduct is a helper function that returns the value of the quantity column for a certain product,specified by the
//id
func getQuantityOfProduct(id int) int{
	var currQuantity int

	stmt := "SELECT quantity FROM clothes WHERE id = ?;"
	row := db.QueryRow(stmt, id)
	err := row.Scan(&currQuantity)
	if err != nil {
		fmt.Println("unsuccessful scanning")
	}
	return currQuantity
}

//isTheStockLow is a function that returns a string with a message, which signalizes that the stock is low if the value of
//the quantity after deletion is below 2
func isTheStockLow(currQuantity int,toDelete int) string{
	if currQuantity-toDelete>=2{
		return ""
	}
	return "The stock is low, you may want to add more products"
}

//isQuantityEnough is a function that returns three values - whether the value of the available quantity is enough to have
//toDelete number removed from it, the quantity available at the moment and a message, received from the isTheStockLow function
//to signalize if the stock is low
func isQuantityEnough(toDelete int, id int) (bool,int,string) {
	currQuantity:=getQuantityOfProduct(id)
	message:=isTheStockLow(currQuantity,toDelete)
	if currQuantity >= toDelete {
		return true,currQuantity,message
	}
	return false,currQuantity,message
}

//the ordersHandler is a function that handles orders and executes them and it is only available for the user that is
//an admin
func ordersHandler(w http.ResponseWriter, r *http.Request) {
	manageAccess(w,r)
	if r.Method == "GET" {
		tpl.ExecuteTemplate(w, "orders.html", nil)
		return
	}

	stmtOrder := "SELECT idorder FROM orderedclothes ORDER BY id DESC LIMIT 1;"
	row := db.QueryRow(stmtOrder)
	var newOrderId int
	err := row.Scan(&newOrderId)
	if err!=nil{
		fmt.Println("unsuccessful scanning")
	}
	newOrderId++

	r.ParseForm()
	var id string 
	var quantity string
	counter:=1 
	for{
		if id!="" && quantity!="" || counter==1{
			id = r.FormValue("idName"+strconv.FormatInt(int64(counter), 10))
			quantity = r.FormValue("quantityName"+strconv.FormatInt(int64(counter), 10))

			idVar, err := strconv.Atoi(id)
			quantVar, err := strconv.Atoi(quantity)

			if quantVar==0{
				break
			}

			flag,oldQuantity,_:=isQuantityEnough(quantVar,idVar)//we can't order a product we don't have 
			if flag==false {
				tpl.ExecuteTemplate(w, "orders.html", "You cannot order this product")
				return
			}

			var insert *sql.Stmt
			insert, err = db.Prepare("INSERT INTO `goproject`.`orderedclothes` (`idorder`,`idclothes`,`quantity`) VALUES (?,?,?);")
			if err != nil {
				fmt.Println("error ordering")
			}
			defer insert.Close()
			_, err = insert.Exec(newOrderId, idVar, quantVar)
			if err != nil {
				fmt.Println("error executing query")
			}
			deleteProducts(oldQuantity,quantVar,idVar)
			counter++
		}else{
			break
		}
	}
	http.Redirect(w, r, "/home", http.StatusFound)
}

//the insertHandler is a function that allows the user, who is an admin, to introduce brand new products
//it receives barcode, id, type, price and the initial quantity of the product via a form in the /insert
//and updates the clothes table in the database  
func insertHandler(w http.ResponseWriter, r *http.Request) {
	manageAccess(w,r)
	if r.Method == "GET" {
		tpl.ExecuteTemplate(w, "insert.html", nil)
		return
	}
	r.ParseForm()
	barcode := r.FormValue("barcodeName")
	id := r.FormValue("idName")
	price := r.FormValue("priceName")
	quantity := r.FormValue("quantityName")
	typeClothes := r.FormValue("typeName")

	var err error

	idVar, err := strconv.Atoi(id)
	priceVar, err := strconv.ParseFloat(price, 32)
	quantVar, err := strconv.Atoi(quantity)

	if barcode == "" || id == "" || price == "" || quantity == "" || typeClothes == "" {
		fmt.Println("everything is empty")
		tpl.ExecuteTemplate(w, "insert.html", nil)
		return
	}

	var insert *sql.Stmt
	insert, err = db.Prepare("INSERT INTO `goproject`.`clothes` (`id`,`type`,`barcode`,`price`,`quantity`) VALUES (?,?,?,?,?);")
	if err != nil {
		//panic
		fmt.Println("error inserting")
	}
	defer insert.Close()
	_, err = insert.Exec(idVar, typeClothes, barcode, priceVar, quantVar)
	if err != nil {
		fmt.Println("error executing query")
		//panic
	}
	http.Redirect(w, r, "/home", http.StatusFound)
}

//the removeHandler function can be used only by an user, who has been checked to be an administrator
//using the function a certain product, specified by its id, can be removed from the clothes table of the database
func removeHandler(w http.ResponseWriter, r *http.Request) {
	manageAccess(w,r)
	if r.Method == "GET" {
		tpl.ExecuteTemplate(w, "remove.html", nil)
		return
	}
	r.ParseForm()
	id := r.FormValue("idName")

	var err error

	idVar, err := strconv.Atoi(id)

	if id == ""  {
		fmt.Println("everything is empty")
		tpl.ExecuteTemplate(w, "remove.html", nil)
		return
	}

	var delete *sql.Stmt
	delete, err = db.Prepare("DELETE FROM `goproject`.`clothes` WHERE (`id` = ?);")
	if err != nil {
		fmt.Println("error deleting")
	}
	defer delete.Close()
	_, err = delete.Exec(idVar)
	if err != nil {
		fmt.Println("error executing query")
	}
	http.Redirect(w, r, "/home", http.StatusFound)
}

//the logoutHandler function deletes the cookie, created upon logging in of the user and thus logs out the user,
//providing him only with the initial log in page 
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	delete(session.Values, "userID")
	session.Save(r, w)
	tpl.ExecuteTemplate(w, "template.html", "Logged Out")
}
