import {describe, expect, it} from 'vitest'
import {transformFilterStringForApi, transformFilterStringFromApi} from '@/helpers/filters'

const nullTitleToIdResolver = (_title: string) => null
const nullIdToTitleResolver = (_id: number) => null
describe('Filter Transformation', () => {

	const fieldCases = {
		'done': 'done',
		'priority': 'priority',
		'percentDone': 'percent_done',
		'dueDate': 'due_date',
		'startDate': 'start_date',
		'endDate': 'end_date',
		'doneAt': 'done_at',
		'reminders': 'reminders',
		'assignees': 'assignees',
		'labels': 'labels',
	}

	describe('For API', () => {
		for (const c in fieldCases) {
			it('should transform all filter params for ' + c + ' to snake_case', () => {
				const transformed = transformFilterStringForApi(c + ' = ipsum', nullTitleToIdResolver, nullTitleToIdResolver)

				expect(transformed).toBe(fieldCases[c] + ' = ipsum')
			})
		}

		it('should correctly resolve labels', () => {
			const transformed = transformFilterStringForApi(
				'labels = lorem',
				(_title: string) => 1,
				nullTitleToIdResolver,
			)

			expect(transformed).toBe('labels = 1')
		})

		const multipleDummyResolver = (_title: string) => {
			switch (_title) {
				case 'lorem':
					return 1
				case 'ipsum':
					return 2
				default:
					return null
			}
		}

		it('should correctly resolve multiple labels', () => {
			const transformed = transformFilterStringForApi(
				'labels = lorem && dueDate = now && labels = ipsum',
				multipleDummyResolver,
				nullTitleToIdResolver,
			)

			expect(transformed).toBe('labels = 1 && due_date = now && labels = 2')
		})

		it('should correctly resolve multiple labels with an in clause', () => {
			const transformed = transformFilterStringForApi(
				'labels in lorem, ipsum && dueDate = now',
				multipleDummyResolver,
				nullTitleToIdResolver,
			)

			expect(transformed).toBe('labels in 1, 2 && due_date = now')
		})
		
		it('should correctly resolve multiple labels with a not in clause', () => {
			const transformed = transformFilterStringForApi(
				'labels not in lorem, ipsum && dueDate = now',
				multipleDummyResolver,
				nullTitleToIdResolver,
			)

			expect(transformed).toBe('labels not in 1, 2 && due_date = now')
		})

		it('should correctly resolve labels with multiple in clauses', () => {
			const transformed = transformFilterStringForApi(
				'labels in lorem || labels in ipsum',
				multipleDummyResolver,
				nullTitleToIdResolver,
			)

			expect(transformed).toBe('labels in 1 || labels in 2')
		})

		it('should correctly resolve projects', () => {
			const transformed = transformFilterStringForApi(
				'project = lorem',
				nullTitleToIdResolver,
				(_title: string) => 1,
			)

			expect(transformed).toBe('project = 1')
		})

		it('should correctly resolve multiple projects', () => {
			const transformed = transformFilterStringForApi(
				'project = lorem && dueDate = now || project = ipsum',
				nullTitleToIdResolver,
				multipleDummyResolver,
			)

			expect(transformed).toBe('project = 1 && due_date = now || project = 2')
		})

		it('should correctly resolve multiple projects with in', () => {
			const transformed = transformFilterStringForApi(
				'project in lorem, ipsum',
				nullTitleToIdResolver,
				multipleDummyResolver,
			)

			expect(transformed).toBe('project in 1, 2')
		})

		it('should resolve projects at the correct position', () => {
			const transformed = transformFilterStringForApi(
				'project = pr',
				nullTitleToIdResolver,
				(_title: string) => 1,
			)

			expect(transformed).toBe('project = 1')
		})

		it('should resolve project and labels independently', () => {
			const transformed = transformFilterStringForApi(
				'project = lorem && labels = ipsum',
				multipleDummyResolver,
				multipleDummyResolver,
			)

			expect(transformed).toBe('project = 1 && labels = 2')
		})

		it('should transform the same attribute multiple times', () => {
			const transformed = transformFilterStringForApi(
				'dueDate = now/_d || dueDate > now/w+1w',
				nullTitleToIdResolver,
				nullTitleToIdResolver,
			)
			
			expect(transformed).toBe('due_date = now/_d || due_date > now/w+1w')
		})
		
		it('should only transform one label occurrence at a time', () => {
			const transformed = transformFilterStringForApi(
				'labels in ipsum || labels in _l',
				multipleDummyResolver,
				nullTitleToIdResolver,
			)

			expect(transformed).toBe('labels in 2 || labels in _l')
		})
		
		it('should correctly transform the cases of fields', () => {
			const transformed = transformFilterStringForApi(
				'startdate > now',
				nullTitleToIdResolver,
				nullTitleToIdResolver,
			)

			expect(transformed).toBe('start_date > now')
		})

		it('should correctly resolve label when the label is called label', () => {
			const transformed = transformFilterStringForApi(
				'labels = label',
				(_title: string) => 1,
				nullTitleToIdResolver,
			)

			expect(transformed).toBe('labels = 1')
		})

		it('should correctly resolve project when the project is called project', () => {
			const transformed = transformFilterStringForApi(
				'project = project',
				nullTitleToIdResolver,
				(_title: string) => 1,
			)

			expect(transformed).toBe('project = 1')
		})

		it('should correctly resolve project in parentheses', () => {
			const transformed = transformFilterStringForApi(
				'( project = Filtertest )',
				nullTitleToIdResolver,
				(_title: string) => _title === 'Filtertest' ? 123 : null,
			)

			expect(transformed).toBe('( project = 123 )')
		})

		it('should correctly resolve project with OR in parentheses', () => {
			const transformed = transformFilterStringForApi(
				'( labels = label || project = Filtertest )',
				(_title: string) => _title === 'label' ? 456 : null,
				(_title: string) => _title === 'Filtertest' ? 123 : null,
			)

			expect(transformed).toBe('( labels = 456 || project = 123 )')
		})
	})

	describe('Special Characters', () => {
		const apostropheResolver = (_title: string) => {
			switch (_title.toLowerCase()) {
				case 'john\'s task':
					return 1
				case 'mary\'s project':
					return 2
				case 'user\'s label':
					return 3
				case 'it\'s working':
					return 4
				default:
					return null
			}
		}

		const apostropheIdResolver = (_id: number) => {
			switch (_id) {
				case 1:
					return 'John\'s Task'
				case 2:
					return 'Mary\'s Project'
				case 3:
					return 'User\'s Label'
				case 4:
					return 'It\'s Working'
				default:
					return null
			}
		}

		describe('Apostrophes in quoted values', () => {
			it('should handle double-quoted labels with apostrophes', () => {
				const transformed = transformFilterStringForApi(
					'labels = "John\'s Task"',
					apostropheResolver,
					nullTitleToIdResolver,
				)

				expect(transformed).toBe('labels = 1')
			})

			it('should handle single-quoted labels with apostrophes', () => {
				const transformed = transformFilterStringForApi(
					'labels = \'Mary\\\'s Project\'',
					apostropheResolver,
					nullTitleToIdResolver,
				)

				expect(transformed).toBe('labels = 2')
			})

			it('should handle projects with apostrophes in double quotes', () => {
				const transformed = transformFilterStringForApi(
					'project = "User\'s Label"',
					nullTitleToIdResolver,
					apostropheResolver,
				)

				expect(transformed).toBe('project = 3')
			})
		})

		describe('Apostrophes in unquoted values', () => {
			it('should handle unquoted labels with apostrophes', () => {
				const transformed = transformFilterStringForApi(
					'labels = John\'s',
					(_title: string) => _title === 'John\'s' ? 1 : null,
					nullTitleToIdResolver,
				)

				expect(transformed).toBe('labels = 1')
			})

			it('should handle unquoted projects with apostrophes', () => {
				const transformed = transformFilterStringForApi(
					'project = Mary\'s',
					nullTitleToIdResolver,
					(_title: string) => _title === 'Mary\'s' ? 2 : null,
				)

				expect(transformed).toBe('project = 2')
			})
		})

		describe('Multiple values with apostrophes', () => {
			it('should handle multiple labels with apostrophes using in operator', () => {
				const transformed = transformFilterStringForApi(
					'labels in "John\'s Task", "Mary\'s Project"',
					apostropheResolver,
					nullTitleToIdResolver,
				)

				expect(transformed).toBe('labels in 1, 2')
			})

			it('should handle multiple labels with apostrophes using not in operator', () => {
				const transformed = transformFilterStringForApi(
					'labels not in "User\'s Label", "It\'s Working"',
					apostropheResolver,
					nullTitleToIdResolver,
				)

				expect(transformed).toBe('labels not in 3, 4')
			})

			it('should handle mixed quoted and unquoted values with apostrophes', () => {
				const mixedResolver = (_title: string) => {
					if (_title === 'John\'s Task') return 1
					if (_title === 'Mary\'s') return 2
					return null
				}

				const transformed = transformFilterStringForApi(
					'labels in "John\'s Task", Mary\'s',
					mixedResolver,
					nullTitleToIdResolver,
				)

				expect(transformed).toBe('labels in 1, 2')
			})
		})

		it('should handle apostrophes in complex filter queries', () => {
			const transformed = transformFilterStringForApi(
				'labels = "John\'s Task" && project = "Mary\'s Project" || priority = 1',
				apostropheResolver,
				apostropheResolver,
			)

			expect(transformed).toBe('labels = 1 && project = 2 || priority = 1')
		})

		describe('Reverse transformation with apostrophes', () => {
			it('should transform labels with apostrophes from API to frontend', () => {
				const transformed = transformFilterStringFromApi(
					'labels = 1',
					apostropheIdResolver,
					nullIdToTitleResolver,
				)

				expect(transformed).toBe('labels = John\'s Task')
			})

			it('should transform projects with apostrophes from API to frontend', () => {
				const transformed = transformFilterStringFromApi(
					'project = 2',
					nullIdToTitleResolver,
					apostropheIdResolver,
				)

				expect(transformed).toBe('project = Mary\'s Project')
			})

			it('should handle multiple values with apostrophes in reverse transformation', () => {
				const transformed = transformFilterStringFromApi(
					'labels in 1, 2',
					apostropheIdResolver,
					nullIdToTitleResolver,
				)

				expect(transformed).toBe('labels in John\'s Task, Mary\'s Project')
			})

			it('should handle complex queries with apostrophes in reverse transformation', () => {
				const transformed = transformFilterStringFromApi(
					'labels = 1 && project = 2 || priority = 1',
					apostropheIdResolver,
					apostropheIdResolver,
				)

				expect(transformed).toBe('labels = John\'s Task && project = Mary\'s Project || priority = 1')
			})
		})
	})

	describe('To API', () => {
		for (const c in fieldCases) {
			it('should transform all filter params for ' + c + ' to snake_case', () => {
				const transformed = transformFilterStringFromApi(fieldCases[c] + ' = ipsum', nullTitleToIdResolver, nullTitleToIdResolver)

				expect(transformed).toBe(c + ' = ipsum')
			})
		}

		it('should correctly resolve labels', () => {
			const transformed = transformFilterStringFromApi(
				'labels = 1',
				(_id: number) => 'lorem',
				nullIdToTitleResolver,
			)

			expect(transformed).toBe('labels = lorem')
		})

		const multipleIdToTitleResolver = (_id: number) => {
			switch (_id) {
				case 1:
					return 'lorem'
				case 2:
					return 'ipsum'
				default:
					return null
			}
		}

		it('should correctly resolve multiple labels', () => {
			const transformed = transformFilterStringFromApi(
				'labels = 1 && due_date = now && labels = 2',
				multipleIdToTitleResolver,
				nullIdToTitleResolver,
			)

			expect(transformed).toBe('labels = lorem && dueDate = now && labels = ipsum')
		})

		it('should correctly resolve multiple labels in', () => {
			const transformed = transformFilterStringFromApi(
				'labels in 1, 2',
				multipleIdToTitleResolver,
				nullIdToTitleResolver,
			)

			expect(transformed).toBe('labels in lorem, ipsum')
		})

		it('should correctly resolve multiple labels in clauses', () => {
			const transformed = transformFilterStringFromApi(
				'labels in 1 || labels in 2',
				multipleIdToTitleResolver,
				nullIdToTitleResolver,
			)

			expect(transformed).toBe('labels in lorem || labels in ipsum')
		})
		
		it('should correctly resolve multiple labels not in', () => {
			const transformed = transformFilterStringFromApi(
				'labels not in 1, 2',
				multipleIdToTitleResolver,
				nullIdToTitleResolver,
			)

			expect(transformed).toBe('labels not in lorem, ipsum')
		})
		
		it('should not touch the label value when it is undefined', () => {
			const transformed = transformFilterStringFromApi(
				'labels = one',
				(_id: number) => undefined,
				nullIdToTitleResolver,
			)

			expect(transformed).toBe('labels = one')
		})

		it('should not touch the label value when it is null', () => {
			const transformed = transformFilterStringFromApi(
				'labels = one',
				(_id: number) => null,
				nullIdToTitleResolver,
			)

			expect(transformed).toBe('labels = one')
		})

		it('should correctly resolve projects', () => {
			const transformed = transformFilterStringFromApi(
				'project = 1',
				nullIdToTitleResolver,
				(_id: number) => 'lorem',
			)

			expect(transformed).toBe('project = lorem')
		})

		it('should correctly resolve multiple projects', () => {
			const transformed = transformFilterStringFromApi(
				'project = 1 && due_date = now || project = 2',
				nullIdToTitleResolver,
				multipleIdToTitleResolver,
			)

			expect(transformed).toBe('project = lorem && dueDate = now || project = ipsum')
		})

		it('should correctly resolve multiple projects in', () => {
			const transformed = transformFilterStringFromApi(
				'project in 1, 2',
				nullIdToTitleResolver,
				multipleIdToTitleResolver,
			)

			expect(transformed).toBe('project in lorem, ipsum')
		})

		it('should not touch the project value when it is undefined', () => {
			const transformed = transformFilterStringFromApi(
				'project = one',
				nullIdToTitleResolver,
				(_id: number) => undefined,
			)

			expect(transformed).toBe('project = one')
		})

		it('should not touch the project value when it is null', () => {
			const transformed = transformFilterStringFromApi(
				'project = one',
				nullIdToTitleResolver,
				(_id: number) => null,
			)

			expect(transformed).toBe('project = one')
		})
		
		it('should transform the same attribute multiple times', () => {
			const transformed = transformFilterStringFromApi(
				'due_date = now/_d || due_date > now/w+1w',
				nullIdToTitleResolver,
				nullIdToTitleResolver,
			)

			expect(transformed).toBe('dueDate = now/_d || dueDate > now/w+1w')
		})

		it('should not replace label _id that appears in other parts of the filter', () => {
			// This tests that position-based replacement is used, not global string replace
			// The label _id "1" should only be replaced in the labels clause, not in "priority = 1"
			const transformed = transformFilterStringFromApi(
				'priority = 1 && labels = 1',
				(_id: number) => _id === 1 ? 'My Label' : null,
				nullIdToTitleResolver,
			)

			expect(transformed).toBe('priority = 1 && labels = My Label')
		})

		it('should not replace project _id that appears in other parts of the filter', () => {
			const transformed = transformFilterStringFromApi(
				'priority = 2 && project = 2',
				nullIdToTitleResolver,
				(_id: number) => _id === 2 ? 'My Project' : null,
			)

			expect(transformed).toBe('priority = 2 && project = My Project')
		})
	})
})
