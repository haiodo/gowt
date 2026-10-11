gowt Windows test kit / Набор для проверки gowt на Windows
==========================================================

ENGLISH
-------
Needs: Windows 10 1809+ or Windows 11, x64 or ARM64. No Go, no install.

1. Unzip the archive to a normal folder (for example C:\gowt-testkit). Do not run from inside the zip.
2. Double-click run.cmd.
3. SmartScreen: the exes are unsigned. If "Windows protected your PC" appears, click "More info",
   then "Run anyway". If the zip itself is blocked: right-click the zip before unzipping,
   Properties, tick "Unblock", OK. An antivirus may quarantine the exes; allow them.
4. Wait. About 3900 tests run; short-lived windows open and close by themselves, and the screen
   may flicker. Expect tens of minutes (not measured yet). Do not use the mouse and keyboard while it
   runs: a click on a test window can fail a test. Browser tests need the WebView2 Runtime
   (present on Windows 11 and most Windows 10).
5. To stop: Ctrl+C in the black window, or close it, or end swttest.exe in Task Manager.
6. When it ends, the window prints the path of the result file:
   result-swttest-<computer>-<arch>.txt (next to run.cmd). Send that one file back.

DPI check (second mode, needs two monitors with different scale, for example 100% and 150%):
1. Run "run.cmd dpi" from a command prompt (or: cmd /c run.cmd dpi).
2. A gallery window opens. Put it on the 100% monitor. Drag it to the 150% monitor.
   Controls, fonts and window size must rescale without restart.
3. Change the scale in Settings - System - Display while the window is open. Same expectation.
4. Close the window and answer the three questions in the black window. Send
   result-dpi-<computer>-<arch>.txt back. (Judging "rescaled" is manual; only the numbers are logged.)

РУССКИЙ
-------
Нужно: Windows 10 1809+ или Windows 11, x64 или ARM64. Go и установка не нужны.

1. Распакуйте архив в обычную папку (например C:\gowt-testkit). Не запускайте из окна архива.
2. Дважды щёлкните run.cmd.
3. SmartScreen: exe не подписаны. Если появилось "Windows защитила ваш компьютер", нажмите
   "Подробнее", затем "Выполнить в любом случае". Если заблокирован сам zip: до распаковки
   правая кнопка на zip, Свойства, галочка "Разблокировать", OK. Антивирус может убрать exe в
   карантин; разрешите их.
4. Ждите. Выполняется около 3900 тестов; короткоживущие окна сами открываются и закрываются,
   экран может мигать. Ожидайте десятки минут (время ещё не измерено). Не трогайте мышь и
   клавиатуру: клик по окну теста может провалить тест. Тестам браузера нужен WebView2 Runtime
   (есть в Windows 11 и в большинстве Windows 10).
5. Остановить: Ctrl+C в чёрном окне, либо закрыть его, либо снять swttest.exe в диспетчере задач.
6. В конце в окне будет путь к файлу результата: result-swttest-<компьютер>-<арх>.txt
   (рядом с run.cmd). Пришлите этот один файл.

Проверка DPI (второй режим, нужны два монитора с разным масштабом, например 100% и 150%):
1. Запустите "run.cmd dpi" из командной строки (или: cmd /c run.cmd dpi).
2. Откроется окно-галерея. Поставьте его на монитор 100%, затем перетащите на монитор 150%.
   Элементы, шрифты и размер окна должны перемасштабироваться без перезапуска.
3. Не закрывая окно, смените масштаб в Параметры - Система - Дисплей. Ожидание то же.
4. Закройте окно и ответьте на три вопроса в чёрном окне. Пришлите result-dpi-<компьютер>-<арх>.txt.
   (Оценка "перемасштабировалось" ручная; в файл пишутся только числа и ваши ответы.)
