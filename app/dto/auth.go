package dto

// LoginRequest 是账号密码登录请求。
type LoginRequest struct {
	Account  string `json:"account" binding:"required,max=50"`
	Password string `json:"password" binding:"required,max=72"`
}

// LoginResponse 是登录成功响应。
type LoginResponse struct {
	Token    string         `json:"token"`
	User     UserResponse   `json:"user"`
	MenuTree []MenuResponse `json:"menuTree"`
}

// ChangePasswordRequest 是登录用户修改密码的请求。
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required,max=72"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=72"`
}
