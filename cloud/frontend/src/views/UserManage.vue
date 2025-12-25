<template>
  <div class="user-manage">
    <div class="page-header">
      <div class="header-content">
        <h1 class="title">用户管理</h1>
        <p class="subtitle">User Management</p>
      </div>
      <div v-if="isAdmin" class="header-actions">
        <el-button type="primary" @click="showCreateDialog">
          <el-icon><Plus /></el-icon>
          新增用户
        </el-button>
      </div>
    </div>

    <!-- 管理员视图：用户列表 -->
    <div v-if="isAdmin" class="content-section">
      <!-- 筛选栏 -->
      <el-card class="filter-card" shadow="never">
        <el-form :inline="true">
          <el-form-item label="角色">
            <el-select v-model="filters.role" placeholder="全部" clearable style="width: 120px">
              <el-option label="管理员" value="admin" />
              <el-option label="普通用户" value="user" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="filters.status" placeholder="全部" clearable style="width: 120px">
              <el-option label="正常" value="active" />
              <el-option label="禁用" value="disabled" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="loadUsers">查询</el-button>
            <el-button @click="resetFilters">重置</el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 用户列表 -->
      <el-card class="table-card" shadow="never">
        <el-table :data="userList" v-loading="loading" style="width: 100%">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="username" label="用户名" min-width="150" />
          <el-table-column prop="email" label="邮箱" min-width="200" />
          <el-table-column prop="role" label="角色" width="120">
            <template #default="{ row }">
              <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'" size="small">
                {{ row.role === 'admin' ? '管理员' : '普通用户' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                {{ row.status === 'active' ? '正常' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="280" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
              <el-button link type="warning" size="small" @click="handleResetPassword(row)">重置密码</el-button>
              <el-button 
                link 
                type="danger" 
                size="small" 
                @click="handleDelete(row)"
                :disabled="row.id === currentUserId"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.page_size"
            :page-sizes="[10, 20, 50, 100]"
            :total="pagination.total"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="loadUsers"
            @current-change="loadUsers"
          />
        </div>
      </el-card>
    </div>

    <!-- 普通用户视图：个人信息 -->
    <div v-else class="content-section">
      <el-card class="profile-card" shadow="never">
        <template #header>
          <div class="card-header">
            <span>个人信息</span>
            <el-button type="primary" size="small" @click="showEditProfileDialog">
              <el-icon><Edit /></el-icon>
              编辑
            </el-button>
          </div>
        </template>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="用户名">
            {{ currentUser?.username }}
          </el-descriptions-item>
          <el-descriptions-item label="邮箱">
            {{ currentUser?.email }}
          </el-descriptions-item>
          <el-descriptions-item label="角色">
            <el-tag :type="currentUser?.role === 'admin' ? 'danger' : 'primary'" size="small">
              {{ currentUser?.role === 'admin' ? '管理员' : '普通用户' }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>
        <div style="margin-top: 20px">
          <el-button type="warning" @click="showChangePasswordDialog">
            <el-icon><Lock /></el-icon>
            修改密码
          </el-button>
        </div>
      </el-card>
    </div>

    <!-- 创建/编辑用户对话框（管理员） -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新增用户' : '编辑用户'"
      width="500px"
    >
      <el-form
        ref="userFormRef"
        :model="userForm"
        :rules="userFormRules"
        label-width="80px"
      >
        <el-form-item label="用户名" prop="username" v-if="dialogMode === 'create'">
          <el-input v-model="userForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="userForm.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="密码" prop="password" v-if="dialogMode === 'create'">
          <el-input v-model="userForm.password" type="password" placeholder="请输入密码" show-password />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="userForm.role" placeholder="请选择角色" style="width: 100%">
            <el-option label="管理员" value="admin" />
            <el-option label="普通用户" value="user" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态" prop="status" v-if="dialogMode === 'edit'">
          <el-select v-model="userForm.status" placeholder="请选择状态" style="width: 100%">
            <el-option label="正常" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 修改密码对话框 -->
    <el-dialog v-model="passwordDialogVisible" title="修改密码" width="500px">
      <el-form
        ref="passwordFormRef"
        :model="passwordForm"
        :rules="passwordFormRules"
        label-width="100px"
      >
        <el-form-item label="原密码" prop="old_password">
          <el-input v-model="passwordForm.old_password" type="password" placeholder="请输入原密码" show-password />
        </el-form-item>
        <el-form-item label="新密码" prop="new_password">
          <el-input v-model="passwordForm.new_password" type="password" placeholder="请输入新密码" show-password />
        </el-form-item>
        <el-form-item label="确认新密码" prop="confirm_password">
          <el-input v-model="passwordForm.confirm_password" type="password" placeholder="请再次输入新密码" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="passwordDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleChangePassword">确定</el-button>
      </template>
    </el-dialog>

    <!-- 编辑个人信息对话框（普通用户） -->
    <el-dialog v-model="profileDialogVisible" title="编辑个人信息" width="500px">
      <el-form
        ref="profileFormRef"
        :model="profileForm"
        :rules="profileFormRules"
        label-width="80px"
      >
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="profileForm.email" placeholder="请输入邮箱" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="profileDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleUpdateProfile">确定</el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框（管理员） -->
    <el-dialog v-model="resetPasswordDialogVisible" title="重置用户密码" width="500px">
      <el-form
        ref="resetPasswordFormRef"
        :model="resetPasswordForm"
        :rules="resetPasswordFormRules"
        label-width="100px"
      >
        <el-form-item label="新密码" prop="new_password">
          <el-input v-model="resetPasswordForm.new_password" type="password" placeholder="请输入新密码" show-password />
        </el-form-item>
        <el-form-item label="确认新密码" prop="confirm_password">
          <el-input v-model="resetPasswordForm.confirm_password" type="password" placeholder="请再次输入新密码" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetPasswordDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleResetPasswordSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Edit, Lock } from '@element-plus/icons-vue'
import { userApi } from '@/api'
import { useAuthStore } from '@/store/auth'
import type { UserInfo } from '@/types/api'

const authStore = useAuthStore()
const currentUser = computed<UserInfo | null>(() => authStore.user as UserInfo | null)
const currentUserId = computed(() => currentUser.value?.id ?? null)
const isAdmin = computed(() => currentUser.value?.role === 'admin')

const loading = ref(false)
const submitLoading = ref(false)
const userList = ref<UserInfo[]>([])
const pagination = reactive({
  page: 1,
  page_size: 10,
  total: 0
})

const filters = reactive({
  role: '',
  status: ''
})

// 用户表单
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const userFormRef = ref<FormInstance>()
const currentEditUserId = ref<number | null>(null)
const userForm = reactive({
  username: '',
  email: '',
  password: '',
  role: 'user' as 'user' | 'admin',
  status: 'active' as 'active' | 'disabled'
})

const validatePassword = (_rule: any, value: any, callback: any) => {
  if (dialogMode.value === 'create' && !value) {
    callback(new Error('请输入密码'))
  } else if (value && !/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,}$/.test(value)) {
    callback(new Error('密码至少8位，且需包含大小写字母和数字'))
  } else {
    callback()
  }
}

const userFormRules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 64, message: '用户名长度在 3 到 64 个字符', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_-]+$/, message: '用户名只能包含字母、数字、下划线和连字符', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ],
  password: [
    { validator: validatePassword, trigger: 'blur' }
  ],
  role: [
    { required: true, message: '请选择角色', trigger: 'change' }
  ]
}

// 修改密码表单
const passwordDialogVisible = ref(false)
const passwordFormRef = ref<FormInstance>()
const passwordForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const validateConfirmPassword = (_rule: any, value: any, callback: any) => {
  if (value === '') {
    callback(new Error('请再次输入密码'))
  } else if (value !== passwordForm.new_password) {
    callback(new Error('两次输入密码不一致'))
  } else {
    callback()
  }
}

const passwordFormRules: FormRules = {
  old_password: [
    { required: true, message: '请输入原密码', trigger: 'blur' }
  ],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, message: '密码至少8位', trigger: 'blur' },
    { pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/, message: '密码必须包含大小写字母和数字', trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

// 编辑个人信息表单
const profileDialogVisible = ref(false)
const profileFormRef = ref<FormInstance>()
const profileForm = reactive({
  email: ''
})

const profileFormRules: FormRules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ]
}

// 重置密码表单（管理员）
const resetPasswordDialogVisible = ref(false)
const resetPasswordFormRef = ref<FormInstance>()
const resetPasswordUserId = ref<number | null>(null)
const resetPasswordForm = reactive({
  new_password: '',
  confirm_password: ''
})

const validateResetConfirmPassword = (_rule: any, value: any, callback: any) => {
  if (value === '') {
    callback(new Error('请再次输入密码'))
  } else if (value !== resetPasswordForm.new_password) {
    callback(new Error('两次输入密码不一致'))
  } else {
    callback()
  }
}

const resetPasswordFormRules: FormRules = {
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, message: '密码至少8位', trigger: 'blur' },
    { pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/, message: '密码必须包含大小写字母和数字', trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, validator: validateResetConfirmPassword, trigger: 'blur' }
  ]
}

// 加载用户列表（管理员）
const loadUsers = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      page_size: pagination.page_size
    }
    if (filters.role) params.role = filters.role
    if (filters.status) params.status = filters.status

    const res = await userApi.listUsers(params)
    userList.value = res.data
    pagination.total = res.total
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '加载用户列表失败')
  } finally {
    loading.value = false
  }
}

// 重置筛选
const resetFilters = () => {
  filters.role = ''
  filters.status = ''
  pagination.page = 1
  loadUsers()
}

// 显示创建对话框
const showCreateDialog = () => {
  dialogMode.value = 'create'
  currentEditUserId.value = null
  Object.assign(userForm, {
    username: '',
    email: '',
    password: '',
    role: 'user',
    status: 'active'
  })
  dialogVisible.value = true
}

// 显示编辑对话框
const handleEdit = (user: UserInfo) => {
  dialogMode.value = 'edit'
  currentEditUserId.value = user.id
  Object.assign(userForm, {
    username: user.username,
    email: user.email,
    password: '',
    role: user.role as 'user' | 'admin',
    status: (user.status || 'active') as 'active' | 'disabled'
  })
  dialogVisible.value = true
}

// 提交表单
const handleSubmit = async () => {
  if (!userFormRef.value) return

  await userFormRef.value.validate(async (valid) => {
    if (!valid) return

    submitLoading.value = true
    try {
      if (dialogMode.value === 'create') {
        await userApi.createUser({
          username: userForm.username,
          email: userForm.email,
          password: userForm.password,
          role: userForm.role
        })
        ElMessage.success('创建成功')
      } else {
        await userApi.updateUser(currentEditUserId.value!, {
          email: userForm.email,
          role: userForm.role,
          status: userForm.status
        })
        ElMessage.success('更新成功')
      }
      dialogVisible.value = false
      loadUsers()
    } catch (error: any) {
      ElMessage.error(error.response?.data?.message || '操作失败')
    } finally {
      submitLoading.value = false
    }
  })
}

// 删除用户
const handleDelete = (user: UserInfo) => {
  ElMessageBox.confirm(
    `确定要删除用户 "${user.username}" 吗？`,
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    try {
      await userApi.deleteUser(user.id)
      ElMessage.success('删除成功')
      loadUsers()
    } catch (error: any) {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  }).catch(() => {
    // 取消删除
  })
}

// 显示修改密码对话框
const showChangePasswordDialog = () => {
  Object.assign(passwordForm, {
    old_password: '',
    new_password: '',
    confirm_password: ''
  })
  passwordDialogVisible.value = true
}

// 修改密码
const handleChangePassword = async () => {
  if (!passwordFormRef.value) return

  await passwordFormRef.value.validate(async (valid) => {
    if (!valid) return

    submitLoading.value = true
    try {
      await userApi.updatePassword({
        old_password: passwordForm.old_password,
        new_password: passwordForm.new_password
      })
      ElMessage.success('密码修改成功')
      passwordDialogVisible.value = false
    } catch (error: any) {
      ElMessage.error(error.response?.data?.message || '密码修改失败')
    } finally {
      submitLoading.value = false
    }
  })
}

// 显示编辑个人信息对话框
const showEditProfileDialog = () => {
  profileForm.email = currentUser.value?.email || ''
  profileDialogVisible.value = true
}

// 更新个人信息
const handleUpdateProfile = async () => {
  if (!profileFormRef.value) return

  await profileFormRef.value.validate(async (valid) => {
    if (!valid) return

    submitLoading.value = true
    try {
      await userApi.updateProfile({
        email: profileForm.email
      })
      ElMessage.success('更新成功')
      profileDialogVisible.value = false
      // 刷新当前用户信息
      const res = await userApi.getProfile()
      authStore.setUser({
        id: res.data.id,
        username: res.data.username,
        email: res.data.email,
        role: res.data.role,
        status: res.data.status
      })
    } catch (error: any) {
      ElMessage.error(error.response?.data?.message || '更新失败')
    } finally {
      submitLoading.value = false
    }
  })
}

// 显示重置密码对话框（管理员）
const handleResetPassword = (user: UserInfo) => {
  resetPasswordUserId.value = user.id
  Object.assign(resetPasswordForm, {
    new_password: '',
    confirm_password: ''
  })
  resetPasswordDialogVisible.value = true
}

// 提交重置密码（管理员）
const handleResetPasswordSubmit = async () => {
  if (!resetPasswordFormRef.value || resetPasswordUserId.value === null) return

  await resetPasswordFormRef.value.validate(async (valid) => {
    if (!valid) return

    submitLoading.value = true
    try {
      await userApi.resetUserPassword(resetPasswordUserId.value!, {
        new_password: resetPasswordForm.new_password
      })
      ElMessage.success('密码重置成功')
      resetPasswordDialogVisible.value = false
    } catch (error: any) {
      ElMessage.error(error.response?.data?.message || '密码重置失败')
    } finally {
      submitLoading.value = false
    }
  })
}

onMounted(() => {
  if (isAdmin.value) {
    loadUsers()
  }
})
</script>

<style scoped>
.user-manage {
  padding: 24px;
  background: #f8fafc;
  min-height: 100vh;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-content .title {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: #1e293b;
  line-height: 1.4;
}

.header-content .subtitle {
  margin: 4px 0 0 0;
  font-size: 14px;
  color: #64748b;
  font-weight: 400;
}

.content-section {
  width: 100%;
}

.filter-card,
.table-card,
.profile-card {
  margin-bottom: 16px;
  border-radius: 12px;
  width: 100%;
}

:deep(.el-card__body) {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

/* ============== 移动端响应式样式 ============== */
@media (max-width: 768px) {
  .user-manage {
    padding: 12px;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .header-content .title {
    font-size: 20px;
  }

  .header-content .subtitle {
    font-size: 13px;
  }

  .header-actions {
    width: 100%;
  }

  .header-actions .el-button {
    width: 100%;
  }

  .filter-card,
  .table-card,
  .profile-card {
    margin-bottom: 12px;
  }

  :deep(.el-card__body) {
    padding: 12px;
  }

  .filter-card :deep(.el-form-item) {
    display: block !important;
    margin-right: 0 !important;
    margin-bottom: 12px !important;
    width: 100% !important;
  }

  .filter-card :deep(.el-form-item__content) {
    width: 100% !important;
  }

  .filter-card :deep(.el-select) {
    width: 100% !important;
  }

  .filter-card :deep(.el-form-item:last-child .el-form-item__content) {
    display: flex;
    gap: 8px;
  }

  .filter-card :deep(.el-form-item:last-child .el-button) {
    flex: 1;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .pagination {
    justify-content: center;
  }

  /* 个人信息卡片优化 */
  .profile-card :deep(.el-descriptions) {
    --el-descriptions-item-bordered-label-background: #f5f7fa;
  }

  .profile-card :deep(.el-descriptions__cell) {
    padding: 10px !important;
  }

  .profile-card :deep(.el-descriptions__label) {
    min-width: 70px !important;
  }
}

@media (max-width: 480px) {
  .user-manage {
    padding: 8px;
  }

  .header-content .title {
    font-size: 18px;
  }

  :deep(.el-card__body) {
    padding: 10px;
  }
}
</style>
