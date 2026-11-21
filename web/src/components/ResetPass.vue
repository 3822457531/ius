<template>
  <div class="reset-pass">
    <el-dialog
        v-model="showDialog"
        :close-on-click-modal="true"
        width="380px"
        :before-close="close"
        :title="t('auth.resetTitle')"
    >
      <div class="form">
        <el-form :model="form" label-width="90px" label-position="left">
          <el-form-item :label="t('auth.resetUsernameLabel')">
            <el-input v-model="form.username" :placeholder="t('auth.resetUsernamePlaceholder')"/>
          </el-form-item>

          <el-form-item :label="t('auth.resetEmailLabel')">
            <el-input v-model="form.mail" :placeholder="t('auth.resetEmailPlaceholder')"/>
          </el-form-item>

          <el-form-item label="验证码">
            <el-row :gutter="12">
              <el-col :span="14">
                <el-input v-model="form.code" maxlength="6" placeholder="邮箱验证码"/>
              </el-col>
              <el-col :span="10">
                <send-msg size="default" :receiver="form.mail"/>
              </el-col>
            </el-row>
          </el-form-item>

          <el-form-item :label="t('auth.resetNewPasswordLabel')">
            <el-input v-model="form.password" type="password"/>
          </el-form-item>
          <el-form-item :label="t('auth.resetRepeatPasswordLabel')">
            <el-input v-model="form.repass" type="password"/>
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="save" round>
            {{ t('auth.resetButton') }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import {computed, ref} from "vue";
import { useI18n } from "vue-i18n";
import SendMsg from "@/components/SendMsg.vue";
import {ElMessage} from "element-plus";
import {httpPost} from "@/utils/http";
import {validateEmail} from "@/utils/validate";

const { t } = useI18n();

const props = defineProps({
  show: Boolean,
  mobile: String
});

const showDialog = computed(() => {
  return props.show
})
const form = ref({
  username: '',
  mail: '',
  code: '',
  password: '',
  repass: ''
})

const emits = defineEmits(['hide']);

const save = () => {
  if (!form.value.username?.trim()) {
    return ElMessage.error(t("auth.resetUsernamePlaceholder"));
  }
  if (!validateEmail(form.value.mail)) {
    return ElMessage.error(t("auth.resetErrorEmailInvalid"));
  }
  if (!form.value.code?.trim()) {
    return ElMessage.error("请输入验证码");
  }
  if (form.value.repass !== form.value.password) {
    return ElMessage.error(t("auth.resetErrorPasswordNotSame"));
  }

  httpPost('/api/user/resetPass', form.value).then(() => {
    ElMessage.success({
      message: t('auth.resetSuccess'), duration: 1000, onClose: () => emits('hide', false)
    })
  }).catch(e => {
    ElMessage.error(t("auth.resetFailPrefix") + e.message);
  })
}

const close = function () {
  emits('hide', false);
}
</script>

<style lang="stylus">
.reset-pass {
  .form {
    padding 10px 40px
  }

  .el-dialog__footer {
    text-align center
  }
}

</style>
