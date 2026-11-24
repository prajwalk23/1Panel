import { watch, onBeforeMount, onMounted, onBeforeUnmount } from 'vue';
import { useRoute } from 'vue-router';
import { MenuStore } from '@/store';
import { DeviceType } from '@/enums/app';
import { useGlobalStore } from '@/composables/useGlobalStore';
/** Matches the Tailwind CSS md breakpoint */
const MOBILE_BREAKPOINT = 768;

/** Re-layout on size change */
export default () => {
    const route = useRoute();
    const { globalStore, isMobile } = useGlobalStore();
    const menuStore = MenuStore();
    const _isMobile = () => {
        const rect = document.body.getBoundingClientRect();
        return rect.width < MOBILE_BREAKPOINT;
    };

    const _syncLayout = () => {
        const isMobileScreen = _isMobile();
        globalStore.toggleDevice(isMobileScreen ? DeviceType.Mobile : DeviceType.Desktop);
        if (isMobileScreen) {
            menuStore.closeSidebar(true);
        }
    };

    const _resizeHandler = () => {
        if (!document.hidden) {
            _syncLayout();
        }
    };

    watch(
        () => route.name,
        () => {
            if (isMobile.value && !menuStore.isCollapse) {
                menuStore.closeSidebar(false);
            }
        },
    );

    onBeforeMount(() => {
        window.addEventListener('resize', _resizeHandler);
    });

    onMounted(() => {
        _syncLayout();
    });

    onBeforeUnmount(() => {
        window.removeEventListener('resize', _resizeHandler);
    });
};
