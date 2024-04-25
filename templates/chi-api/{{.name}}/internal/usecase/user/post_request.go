package user

type postRequest struct {
	Name  string `validate:"required|min_len:7" message:"required:{field} is required"`
	Email string `validate:"email" message:"email is invalid"`
	Age   int    `validate:"required|int|min:18|max:99" message:"int:age must int|min:age min value is 18|max:age max value is 99"`
}
