import {i18n} from '@/i18n'
import type {IProject} from '@/modelTypes/IProject'
import {useCompanyStore} from '@/stores/company'
import type {ICompanyRelation} from '@/services/company'

export function getProjectTitle(project: IProject) {
	if (project.id === -1) {
		return i18n.global.t('project.pseudo.favorites.title')
	}

	if (project.title === 'Inbox') {
		return i18n.global.t('project.inboxTitle')
	}

	const companyStore = useCompanyStore()
	const relations: ICompanyRelation[] = companyStore.relations || []

	const relation = relations.find(r => r.project_id === project.id)
	if (relation) {
		return `from ${relation.superior_username}`
	}

	return project.title
}
