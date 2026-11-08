import {test, expect} from 'vitest';
import {pathBasename, pathDirname, pathDisplayDir} from '../src/lib/utils/path';

test('path helpers handle POSIX and Windows separators', () => {
    expect(pathBasename('/home/lain/Pictures/wall.jpg')).toBe('wall.jpg');
    expect(pathBasename('C:\\Users\\enrell\\Pictures\\wall.jpg')).toBe(
        'wall.jpg'
    );
    expect(pathBasename('wall.jpg')).toBe('wall.jpg');

    expect(pathDirname('/home/lain/Pictures/wall.jpg')).toBe(
        '/home/lain/Pictures'
    );
    expect(pathDirname('C:\\Users\\enrell\\Pictures\\wall.jpg')).toBe(
        'C:\\Users\\enrell\\Pictures'
    );
    expect(pathDirname('C:\\Users\\enrell\\Pictures\\')).toBe(
        'C:\\Users\\enrell\\Pictures'
    );
    expect(pathDirname('/wall.jpg')).toBe('/');
    expect(pathDirname('wall.jpg')).toBe('');
});

test('the home prefix is shortened on every platform', () => {
    expect(pathDisplayDir('/home/lain/Pictures/wall.jpg')).toBe('~/Pictures');
    expect(pathDisplayDir('/Users/lain/Pictures/wall.jpg')).toBe('~/Pictures');
    expect(pathDisplayDir('C:\\Users\\enrell\\Pictures\\wall.jpg')).toBe(
        '~\\Pictures'
    );
    expect(pathDisplayDir('D:\\Wallpapers\\wall.jpg')).toBe('D:\\Wallpapers');
});
