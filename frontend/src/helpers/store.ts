import { defineStore } from 'pinia';
import type { Info, Session, SessionData } from '@/helpers';

type State = {
    info?: Info
    session: Session
}

export const useStore = defineStore('state', {
    state: (): State => {
        return {
            session: { links: {} }
        };
    },
    actions: {
        /**
         * Clear the current section
         */
        clearSession() {
            this.session = { id: null, links: {} };
        },

        /**
         * Update our current session with fresh data
         * @param data The data to update it with
         * @param options A list of options
         */
        updateSession(data: SessionData, options?: { fullSession?: boolean }) {
            this.session.packages = data.packages;
            this.session.overall_state = data.overall_state;

            if (options?.fullSession) {
                this.session.id = data.id;
                this.session.links = data.links;
            }
        }
    }
});
