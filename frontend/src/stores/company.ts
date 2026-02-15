import {ref, computed} from 'vue'
import {defineStore, acceptHMRUpdate} from 'pinia'
import CompanyService from '@/services/company'
import type {ICompany} from '@/modelTypes/ICompany'
import type {ICompanyRelation} from '@/services/company'

export const useCompanyStore = defineStore('company', () => {
	const companies = ref<ICompany[]>([])
	const currentCompanyId = ref<number | null>(null)
	const isLoading = ref(false)
	const error = ref<string | null>(null)
	const relations = ref<ICompanyRelation[]>([])

	const companyService = new CompanyService()

	const currentCompany = computed(() => {
		if (!currentCompanyId.value) {
			return null
		}
		return companies.value.find(c => c.id === currentCompanyId.value) || null
	})

	async function loadCompanies() {
		if (isLoading.value) return
		isLoading.value = true
		error.value = null
		try {
			companies.value = await companyService.getCompanies()
			if (companies.value.length > 0) {
				const savedCompanyId = localStorage.getItem('vikunja-current-company-id')
				if (savedCompanyId) {
					const savedId = parseInt(savedCompanyId, 10)
					const exists = companies.value.find(c => c.id === savedId)
					if (exists) {
						currentCompanyId.value = savedId
					} else {
						currentCompanyId.value = companies.value[0].id
					}
				} else {
					currentCompanyId.value = companies.value[0].id
				}
			}
		} catch (err: any) {
			error.value = err?.message || 'Failed to load companies'
			console.error('[Company] Failed to load companies:', err)
		} finally {
			isLoading.value = false
		}
	}

	async function loadRelationsAsSubordinate() {
		try {
			relations.value = await companyService.getRelationsAsSubordinate()
		} catch (err: any) {
			console.error('[Company] Failed to load relations:', err)
		}
	}

	function setCurrentCompany(companyId: number) {
		currentCompanyId.value = companyId
		localStorage.setItem('vikunja-current-company-id', String(companyId))
	}

	return {
		companies,
		currentCompanyId,
		currentCompany,
		isLoading,
		error,
		relations,
		loadCompanies,
		loadRelationsAsSubordinate,
		setCurrentCompany,
	}
})

if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useCompanyStore, import.meta.hot))
}
