export const version = '1.0';

// Errors sent by the server
export const errors = {
    NO_SUITABLE_VERSION: 'no_suitable_version',
};

export const fixableErrors = [ errors.NO_SUITABLE_VERSION ];

export const messages = {
    // Messages sent to the server
    PRELIMINARY: 'preliminary',
    TOGGLE_LINK: 'toggle_link',
    PROCESS: 'process',
    PACKAGE: 'package',
    GET_DOWNLOAD: 'get_download',
    DELETE: 'delete',

    // Messages sent from the server
    CONNECTED: 'connected',
    PRELIMINARY_START: 'preliminary_start',
    PRELIMINARY_STEP: 'preliminary_step',
    PRELIMINARY_DONE: 'preliminary_done',
    PROCESS_START: 'process_start',
    PROCESS_STEP: 'process_step',
    PROCESS_DONE: 'process_done',
    PACKAGE_START: 'package_start',
    PACKAGE_DONE: 'package_done',
    GET_DOWNLOAD_START: 'get_download_start',
    GET_DOWNLOAD_DONE: 'get_download_done',
    DELETED: 'deleted'
};
