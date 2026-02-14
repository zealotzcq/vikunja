import {AuthenticatedHTTPFactory} from '@/helpers/fetcher'
import {getToken} from '@/helpers/auth'
import type {ICompany} from '@/modelTypes/ICompany'

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
}
