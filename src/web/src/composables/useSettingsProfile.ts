import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import {
  changePassword, uploadAvatar, deleteAvatar, updateProfile,
  validateNumisBidsCredentials, testPushover,
} from '@/api/client'

export function useSettingsProfile() {
  const auth = useAuthStore()

  // Avatar
  const avatarUrl = ref('/coin-logo.jpg')

  function updateAvatarUrl() {
    avatarUrl.value = auth.user?.avatarPath ? `/uploads/${auth.user.avatarPath}` : '/coin-logo.jpg'
  }
  updateAvatarUrl()

  async function handleAvatarUpload(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (!file) return
    try {
      const res = await uploadAvatar(file)
      if (auth.user) {
        auth.user.avatarPath = res.data.avatarPath
        localStorage.setItem('user', JSON.stringify(auth.user))
      }
      updateAvatarUrl()
    } catch { /* ignore */ }
  }

  async function handleAvatarDelete() {
    try {
      await deleteAvatar()
      if (auth.user) {
        auth.user.avatarPath = ''
        localStorage.setItem('user', JSON.stringify(auth.user))
      }
      updateAvatarUrl()
    } catch { /* ignore */ }
  }

  // Profile
  const profileEmail = ref(auth.user?.email || '')
  const profileBio = ref(auth.user?.bio || '')
  const profileZipCode = ref(auth.user?.zipCode || '')
  const nbUsername = ref(auth.user?.numisBidsUsername || '')
  const nbPassword = ref('')
  const cngUsername = ref(auth.user?.cngUsername || '')
  const cngPassword = ref('')
  const pushoverKey = ref('')
  const pushoverTesting = ref(false)
  const pushoverTestMsg = ref('')
  const pushoverTestError = ref(false)
  const profilePublic = ref(auth.user?.isPublic || false)
  const coinOfDayEnabled = ref(auth.user?.coinOfDayEnabled ?? true)
  const emperorTrackerEnabled = ref(auth.user?.emperorTrackerEnabled ?? false)
  const emperorTrackerShowUsurpers = ref(auth.user?.emperorTrackerShowUsurpers ?? false)
  const emperorTrackerShowEmpresses = ref(auth.user?.emperorTrackerShowEmpresses ?? false)
  const emperorTrackerShowOtherFigures = ref(auth.user?.emperorTrackerShowOtherFigures ?? false)
  const profileMsg = ref('')
  const profileError = ref(false)
  const profileSaving = ref(false)
  const showPrivacyWarning = ref(false)

  function onPublicToggle(e: Event) {
    const checked = (e.target as HTMLInputElement).checked
    if (!checked && profilePublic.value) {
      ;(e.target as HTMLInputElement).checked = true
      showPrivacyWarning.value = true
    } else {
      profilePublic.value = checked
    }
  }

  function confirmGoPrivate() {
    profilePublic.value = false
    showPrivacyWarning.value = false
  }

  function cancelGoPrivate() {
    showPrivacyWarning.value = false
  }

  // NumisBids validation
  const nbValidating = ref(false)
  const nbValidationError = ref('')
  const cngValidating = ref(false)
  const cngValidationError = ref('')

  async function handleSaveProfile() {
    profileMsg.value = ''
    profileError.value = false
    profileSaving.value = true
    nbValidationError.value = ''
    cngValidationError.value = ''
    try {
      if (nbPassword.value && nbUsername.value) {
        nbValidating.value = true
        try {
          const valRes = await validateNumisBidsCredentials(nbUsername.value, nbPassword.value)
          if (!valRes.data.valid) {
            nbValidationError.value = valRes.data.error || 'Invalid NumisBids credentials'
            profileSaving.value = false
            nbValidating.value = false
            return
          }
        } catch {
          nbValidationError.value = 'Could not validate NumisBids credentials. Check your username and password.'
          profileSaving.value = false
          nbValidating.value = false
          return
        } finally {
          nbValidating.value = false
        }
      }
      if (cngPassword.value && cngUsername.value) {
        cngValidating.value = true
        try {
          const valRes = await validateNumisBidsCredentials(cngUsername.value, cngPassword.value, 'cng')
          if (!valRes.data.valid) {
            cngValidationError.value = valRes.data.error || 'Invalid CNG credentials'
            profileSaving.value = false
            cngValidating.value = false
            return
          }
        } catch {
          cngValidationError.value = 'Could not validate CNG credentials. Check your username and password.'
          profileSaving.value = false
          cngValidating.value = false
          return
        } finally {
          cngValidating.value = false
        }
      }

      const data: Record<string, unknown> = {
        email: profileEmail.value,
        bio: profileBio.value,
        zipCode: profileZipCode.value,
        isPublic: profilePublic.value,
        numisBidsUsername: nbUsername.value,
        cngUsername: cngUsername.value,
        coinOfDayEnabled: coinOfDayEnabled.value,
        emperorTrackerEnabled: emperorTrackerEnabled.value,
        emperorTrackerShowUsurpers: emperorTrackerShowUsurpers.value,
        emperorTrackerShowEmpresses: emperorTrackerShowEmpresses.value,
        emperorTrackerShowOtherFigures: emperorTrackerShowOtherFigures.value,
      }
      if (nbPassword.value) {
        data.numisBidsPassword = nbPassword.value
      }
      if (cngPassword.value) {
        data.cngPassword = cngPassword.value
      }
      if (pushoverKey.value !== '') {
        data.pushoverUserKey = pushoverKey.value
      }
      const res = await updateProfile(data as Parameters<typeof updateProfile>[0])
      if (auth.user) {
        auth.user.email = res.data.email
        auth.user.bio = res.data.bio
        auth.user.zipCode = res.data.zipCode
        auth.user.isPublic = res.data.isPublic
        auth.user.numisBidsUsername = res.data.numisBidsUsername
        auth.user.numisBidsConfigured = res.data.numisBidsConfigured
        auth.user.cngUsername = res.data.cngUsername
        auth.user.cngConfigured = res.data.cngConfigured
        auth.user.pushoverEnabled = res.data.pushoverEnabled
        auth.user.coinOfDayEnabled = res.data.coinOfDayEnabled
        auth.user.emperorTrackerEnabled = res.data.emperorTrackerEnabled
        auth.user.emperorTrackerShowUsurpers = res.data.emperorTrackerShowUsurpers
        auth.user.emperorTrackerShowEmpresses = res.data.emperorTrackerShowEmpresses
        auth.user.emperorTrackerShowOtherFigures = res.data.emperorTrackerShowOtherFigures
        localStorage.setItem('user', JSON.stringify(auth.user))
      }
      nbPassword.value = ''
      cngPassword.value = ''
      pushoverKey.value = ''
      profileMsg.value = 'Profile saved'
    } catch {
      profileMsg.value = 'Failed to save profile'
      profileError.value = true
    } finally {
      profileSaving.value = false
    }
  }

  // Password
  const currentPassword = ref('')
  const newPassword = ref('')
  const confirmPassword = ref('')
  const passwordMsg = ref('')
  const passwordError = ref(false)
  const passwordLoading = ref(false)

  async function handleChangePassword() {
    passwordMsg.value = ''
    passwordError.value = false

    if (newPassword.value !== confirmPassword.value) {
      passwordMsg.value = 'New passwords do not match'
      passwordError.value = true
      return
    }

    passwordLoading.value = true
    try {
      await changePassword(currentPassword.value, newPassword.value)
      passwordMsg.value = 'Password changed successfully'
      currentPassword.value = ''
      newPassword.value = ''
      confirmPassword.value = ''
    } catch {
      passwordMsg.value = 'Failed — check your current password'
      passwordError.value = true
    } finally {
      passwordLoading.value = false
    }
  }

  // Pushover test
  async function handleTestPushover() {
    pushoverTesting.value = true
    pushoverTestMsg.value = ''
    pushoverTestError.value = false
    try {
      await testPushover()
      pushoverTestMsg.value = 'Test notification sent'
    } catch {
      pushoverTestMsg.value = 'Failed to send test notification'
      pushoverTestError.value = true
    } finally {
      pushoverTesting.value = false
    }
  }

  return {
    // Avatar
    avatarUrl,
    handleAvatarUpload,
    handleAvatarDelete,
    // Profile
    profileEmail,
    profileBio,
    profileZipCode,
    nbUsername,
    nbPassword,
    cngUsername,
    cngPassword,
    pushoverKey,
    pushoverTesting,
    pushoverTestMsg,
    pushoverTestError,
    handleTestPushover,
    profilePublic,
    profileMsg,
    profileError,
    profileSaving,
    showPrivacyWarning,
    onPublicToggle,
    confirmGoPrivate,
    cancelGoPrivate,
    nbValidating,
    nbValidationError,
    cngValidating,
    cngValidationError,
    handleSaveProfile,
    coinOfDayEnabled,
    emperorTrackerEnabled,
    emperorTrackerShowUsurpers,
    emperorTrackerShowEmpresses,
    emperorTrackerShowOtherFigures,
    // Password
    currentPassword,
    newPassword,
    confirmPassword,
    passwordMsg,
    passwordError,
    passwordLoading,
    handleChangePassword,
  }
}
