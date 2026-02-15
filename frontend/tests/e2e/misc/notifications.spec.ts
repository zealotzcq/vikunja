import {test, expect} from '../../support/fixtures'

test.describe('Duplicate Notifications', () => {
	test('Merges duplicate notifications and shows count', async ({authenticatedPage: page}) => {
		await page.goto('/')
		await page.waitForLoadState('networkidle')

		// Trigger the same notification twice via the Vue app using $notify directly
		await page.evaluate(() => {
			const app = document.getElementById('app') as unknown as { __vue_app__: { config: { globalProperties: { $notify: (notification: unknown) => void } } } }
			const vueApp = app.__vue_app__
			vueApp.config.globalProperties.$notify({
				type: 'success',
				title: 'Test',
				text: 'Duplicate Test',
			})
			vueApp.config.globalProperties.$notify({
				type: 'success',
				title: 'Test',
				text: 'Duplicate Test',
			})
		})

		// Should only show one notification with a count of ×2
		const notification = page.locator('.global-notification .vue-notification.success')
		await expect(notification).toHaveCount(1)
		await expect(notification.locator('.notification-content')).toContainText('Duplicate Test')
		await expect(notification.locator('.notification-content')).toContainText('×2')
	})
})
