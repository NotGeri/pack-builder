/**
 * Sleep X ms before resolving the promise
 * @param time The time in ms
 */
export const sleep = (time: number) => {
    return new Promise(resolve => {
        setTimeout(resolve, time);
    });
};
