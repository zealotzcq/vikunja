import {AuthenticatedHTTPFactory} from '@/helpers/fetcher'
import {getToken} from '@/helpers/auth'
import type {ICompany} from '@/modelTypes/ICompany'

export interface ICompanyRelation {
	id: number
	company_id: number
	superior_user_id: number
	subordinate_user_id: number
	project_id: number
	created: number
	updated: number
	superior_username: string
}

export interface ICompanyProjectMap {
	company_id: number
	project_ids: number[]
}

export default class CompanyService {
	http = AuthenticatedHTTPFactory()

	async getCompanies(): Promise<ICompany[]> {
		const token = getToken()
		if (!token) {
			throw new Error('No authentication token available')
		}

		const response = await this.http.get('/companies')
		return response.data
	}

	async getRelationsAsSubordinate(): Promise<ICompanyRelation[]> {
		const response = await this.http.get('/companies/relations/subordinate')
		return response.data as ICompanyRelation[]
	}

	async getCompanyProjectMap(): Promise<ICompanyProjectMap[]> {
		const response = await this.http.get('/companies/project-map')
		return response.data as ICompanyProjectMap[]
	}
}
