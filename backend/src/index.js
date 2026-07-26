import { Plugins } from '@capacitor/core';
const { App, StatusBar, SplashScreen } = Plugins;
async function initApp() {
    await App.addListener('appStateChange', ({ isActive }) => {
        if (isActive) {
            // App is active
        }
    });
    await StatusBar.setStyle({
        style: 'dark',
        translucent: false,
    });
    await SplashScreen.hide();
}
initApp();
//# sourceMappingURL=index.js.map