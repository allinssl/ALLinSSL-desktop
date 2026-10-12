import { Transition, type Component as ComponentType, h } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AllinSslThemeProvider from '@/components/AllinSslThemeProvider'
import DesktopTitleBar from '@/components/DesktopTitleBar'
 

export default defineComponent({
	name: 'App',
	setup() {
		const route = useRoute()
		return () => (
			<AllinSslThemeProvider>
				{!route.matched.some(record => record.name === 'layout') && <DesktopTitleBar class="fixed top-0 left-0 w-full z-[1000]" />}
				<RouterView>
					{({ Component }: { Component: ComponentType }) => (
						<Transition name="route-slide" mode="out-in">
							{Component && h(Component)}
						</Transition>
					)}
				</RouterView>
			</AllinSslThemeProvider>
		)
	},
})
