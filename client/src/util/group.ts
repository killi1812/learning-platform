import type { UserClaims } from '@/types/auth'
import type { Group, GroupMember } from '@/types/course'

export function isCurrentUserGroupMember(
  member: Pick<GroupMember, 'uuid' | 'email'>,
  user?: Pick<UserClaims, 'userUuid' | 'email'>,
): boolean {
  if (!user) {
    return false
  }
  if (user.userUuid && member.uuid === user.userUuid) {
    return true
  }
  const userEmail = user.email?.toLowerCase()
  return Boolean(userEmail && member.email?.toLowerCase() === userEmail)
}

export function findUserGroup(
  groups: readonly Group[],
  user?: Pick<UserClaims, 'userUuid' | 'email'>,
): Group | undefined {
  if (!user || groups.length === 0) {
    return undefined
  }
  return groups.find(group => group.members?.some(member => isCurrentUserGroupMember(member, user)))
}
